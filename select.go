package lorm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/samber/lo"

	"github.com/yvvlee/lorm/builder"
)

func newSelectBuilder[T any](engine *Engine) (*builder.SelectBuilder, bool) {
	var t T
	model, isModel := any(t).(Model)
	selectBuilder := new(builder.SelectBuilder)
	if !isModel {
		return selectBuilder, false
	}
	fields := model.LormModelDescriptor().AllFields()
	if escaper := engine.Escaper(); escaper != nil {
		fields = lo.Map(fields, func(field string, _ int) string {
			return escaper.Escape(field)
		})
	}
	selectBuilder.Select(fields...)
	if table, ok := any(t).(Table); ok {
		selectBuilder.From(engine.Escaper().Escape(table.TableName()))
	}
	return selectBuilder, true
}

// Select builds a SELECT statement that scans rows into values of T.
// Model result types receive their generated columns and table automatically.
// Other result types are scanned from exactly one explicitly selected column.
func (e *Engine) Select[T any]() *SelectStmt[T] {
	selectBuilder, modelResult := newSelectBuilder[T](e)
	return &SelectStmt[T]{
		engine:      e,
		builder:     selectBuilder,
		modelResult: modelResult,
	}
}

// SelectStmt is a fluent SELECT builder that scans rows into values of T.
type SelectStmt[T any] struct {
	engine      *Engine
	builder     *builder.SelectBuilder
	modelResult bool
	err         error
}

func (s *SelectStmt[T]) reset() {
	s.builder, s.modelResult = newSelectBuilder[T](s.engine)
	s.err = nil
}

// Clone returns a copy of the statement state. Terminal methods still reset
// only the statement they are called on.
func (s *SelectStmt[T]) Clone() *SelectStmt[T] {
	return &SelectStmt[T]{
		engine:      s.engine,
		builder:     s.builder.Clone(),
		modelResult: s.modelResult,
		err:         s.err,
	}
}

// Get returns the first matching value and whether a row was found.
func (s *SelectStmt[T]) Get(ctx context.Context) (T, bool, error) {
	var t T
	defer s.reset()
	if s.err != nil {
		return t, false, s.err
	}
	query, args, err := s.builder.Clone().Limit(1).ToSql()
	if err != nil {
		return t, false, err
	}
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		return t, false, err
	}
	defer rows.Close()
	res, err := scanSelectValue[T](rows, s.modelResult)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, false, nil
		}
		return t, false, err
	}
	return res, true, nil
}

// Exist reports whether the query returns at least one row.
func (s *SelectStmt[T]) Exist(ctx context.Context) (bool, error) {
	defer s.reset()
	if s.err != nil {
		return false, s.err
	}
	query, args, err := s.builder.Clone().Select("1").Limit(1).ToSql()
	if err != nil {
		return false, err
	}
	return s.engine.Exist(ctx, query, args...)
}

// Find returns all matching values.
func (s *SelectStmt[T]) Find(ctx context.Context) ([]T, error) {
	defer s.reset()
	if s.err != nil {
		return nil, s.err
	}
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	values, err := scanSelectValues[T](rows, s.modelResult)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// Page returns the requested page of results together with the total row count.
func (s *SelectStmt[T]) Page(ctx context.Context, page, size uint64) ([]T, uint64, error) {
	defer s.reset()
	if s.err != nil {
		return nil, 0, s.err
	}
	if size == 0 {
		return nil, 0, errors.New("size can not be zero")
	}
	if page == 0 {
		page = 1
	}
	pageIndex := page - 1
	offsetOverflow := pageIndex > ^uint64(0)/size
	var offset uint64
	if !offsetOverflow {
		offset = pageIndex * size
	}
	countStmt := s.engine.Select[uint64]()
	// Count with a derived builder so filters stay in sync with the data query.
	countStmt.builder = s.builder.ToCountBuilder()
	count, ok, err := countStmt.Get(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !ok || count == 0 {
		return nil, 0, nil
	}
	if offsetOverflow || offset >= count {
		return nil, count, nil
	}
	s.builder.Limit(size).Offset(offset)
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, count, err
	}
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, count, nil
		}
		return nil, count, err
	}
	defer rows.Close()
	list, err := scanSelectValues[T](rows, s.modelResult)
	if err != nil {
		return nil, count, err
	}
	return list, count, nil
}

// Prefix adds an expression to the beginning of the query
func (s *SelectStmt[T]) Prefix(sql string, args ...any) *SelectStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *SelectStmt[T]) PrefixExpr(expr builder.Sqlizer) *SelectStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Distinct adds a DISTINCT clause to the query.
func (s *SelectStmt[T]) Distinct() *SelectStmt[T] {
	s.builder.Distinct()
	return s
}

// Options adds select option to the query
func (s *SelectStmt[T]) Options(options ...string) *SelectStmt[T] {
	s.builder.Options(options...)
	return s
}

// Select set result columns to the query.
func (s *SelectStmt[T]) Select(columns ...string) *SelectStmt[T] {
	s.builder.Select(columns...)
	return s
}

// AddColumn adds a result column to the query.
// Unlike Select, AddColumn accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *SelectStmt[T]) AddColumn(column any, args ...any) *SelectStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// RemoveColumns remove all columns from query.
// Must add a new column with Column or Select methods, otherwise
// return a error.
func (s *SelectStmt[T]) RemoveColumns() *SelectStmt[T] {
	s.builder.RemoveColumns()
	return s
}

// Column adds a result column to the query.
// Unlike Select, Column accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *SelectStmt[T]) Column(column any, args ...any) *SelectStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// From sets the FROM clause of the query.
func (s *SelectStmt[T]) From(from string) *SelectStmt[T] {
	s.builder.From(from)
	return s
}

// FromSelect sets a subquery into the FROM clause of the query.
func (s *SelectStmt[T]) FromSelect(from *builder.SelectBuilder, alias string) *SelectStmt[T] {
	s.builder.FromSelect(from, alias)
	return s
}

// JoinClause adds a join clause to the query.
func (s *SelectStmt[T]) JoinClause(pred any, args ...any) *SelectStmt[T] {
	s.builder.JoinClause(pred, args...)
	return s
}

// Join adds a JOIN clause to the query.
func (s *SelectStmt[T]) Join(join string, rest ...any) *SelectStmt[T] {
	s.builder.Join(join, rest...)
	return s
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (s *SelectStmt[T]) LeftJoin(join string, rest ...any) *SelectStmt[T] {
	s.builder.LeftJoin(join, rest...)
	return s
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (s *SelectStmt[T]) RightJoin(join string, rest ...any) *SelectStmt[T] {
	s.builder.RightJoin(join, rest...)
	return s
}

// InnerJoin adds a INNER JOIN clause to the query.
func (s *SelectStmt[T]) InnerJoin(join string, rest ...any) *SelectStmt[T] {
	s.builder.InnerJoin(join, rest...)
	return s
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (s *SelectStmt[T]) CrossJoin(join string, rest ...any) *SelectStmt[T] {
	s.builder.CrossJoin(join, rest...)
	return s
}

// Where adds an expression to the WHERE clause of the query.
//
// Expressions are ANDed together in the generated SQL.
//
// Where accepts several types for its pred argument:
//
// nil OR "" - ignored.
//
// string - SQL expression.
// If the expression has SQL placeholders then a set of arguments must be passed
// as well, one for each placeholder.
//
// map[string]any OR Eq - map of SQL expressions to values. Each key is
// transformed into an expression like "<key> = ?", with the corresponding value
// bound to the placeholder. Nil, slices, arrays, pointers, and driver values
// are passed as one bound value; use builder.IsNull, builder.IsNotNull,
// builder.In, or builder.NotIn explicitly when you need those predicate forms.
//
// Where will panic if pred isn't any of the above types.
func (s *SelectStmt[T]) Where(pred any, args ...any) *SelectStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate derived from the model metadata.
func (s *SelectStmt[T]) ID(id any) *SelectStmt[T] {
	if s.err != nil {
		return s
	}
	var t T
	model, ok := any(t).(Model)
	if !ok {
		s.err = errors.New("lorm.Engine.Select().ID() requires a Model result type")
		return s
	}
	primaryKeys := model.LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		s.err = errors.New("lorm.Engine.Select().ID() only supports models with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{s.engine.Escaper().Escape(primaryKeys[0]): id})
	return s
}

// GroupBy adds GROUP BY expressions to the query.
func (s *SelectStmt[T]) GroupBy(groupBys ...string) *SelectStmt[T] {
	s.builder.GroupBy(groupBys...)
	return s
}

// Having adds an expression to the HAVING clause of the query.
//
// See Where.
func (s *SelectStmt[T]) Having(pred any, rest ...any) *SelectStmt[T] {
	s.builder.Having(escapePredicate(s.engine.Escaper(), pred), rest...)
	return s
}

// OrderByClause adds ORDER BY clause to the query.
func (s *SelectStmt[T]) OrderByClause(pred any, args ...any) *SelectStmt[T] {
	s.builder.OrderByClause(pred, args...)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *SelectStmt[T]) OrderBy(orderBys ...string) *SelectStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *SelectStmt[T]) Limit(limit uint64) *SelectStmt[T] {
	s.builder.Limit(limit)
	return s
}

// RemoveLimit removes the LIMIT clause.
func (s *SelectStmt[T]) RemoveLimit() *SelectStmt[T] {
	s.builder.RemoveLimit()
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *SelectStmt[T]) Offset(offset uint64) *SelectStmt[T] {
	s.builder.Offset(offset)
	return s
}

// RemoveOffset removes OFFSET clause.
func (s *SelectStmt[T]) RemoveOffset() *SelectStmt[T] {
	s.builder.RemoveOffset()
	return s
}

// Suffix adds an expression to the end of the query
func (s *SelectStmt[T]) Suffix(sql string, args ...any) *SelectStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *SelectStmt[T]) SuffixExpr(expr builder.Sqlizer) *SelectStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
