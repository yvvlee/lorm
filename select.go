package lorm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/samber/lo"

	"github.com/yvvlee/lorm/builder"
)

func newQueryModelBuilder[T Model](engine *Engine) *builder.SelectBuilder {
	var t T
	fields := t.LormModelDescriptor().AllFields()
	if escaper := engine.Escaper(); escaper != nil {
		fields = lo.Map(fields, func(field string, _ int) string {
			return escaper.Escape(field)
		})
	}
	selectBuilder := new(builder.SelectBuilder)
	selectBuilder.Select(fields...)
	if table, ok := any(t).(Table); ok {
		selectBuilder.From(engine.Escaper().Escape(table.TableName()))
	}
	return selectBuilder
}

// Query builds a SELECT statement that scans rows into models of T.
func Query[T Model](engine *Engine) *QueryModelStmt[T] {
	return &QueryModelStmt[T]{
		engine:  engine,
		builder: newQueryModelBuilder[T](engine),
	}
}

// QueryModelStmt is a fluent SELECT builder that scans rows into models of T.
type QueryModelStmt[T Model] struct {
	engine  *Engine
	builder *builder.SelectBuilder
	err     error
}

func (s *QueryModelStmt[T]) reset() {
	s.builder = newQueryModelBuilder[T](s.engine)
	s.err = nil
}

// Get returns the first matching model or the zero value when no row matches.
func (s *QueryModelStmt[T]) Get(ctx context.Context) (T, error) {
	var t T
	defer s.reset()
	if s.err != nil {
		return t, s.err
	}
	query, args, err := s.builder.ToSql()
	if err != nil {
		return t, err
	}
	res := t.New()
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	if err = ScanModel(rows, res); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, nil
		}
		return t, err
	}
	return res.(T), nil
}

// Exist reports whether the query returns at least one row.
func (s *QueryModelStmt[T]) Exist(ctx context.Context) (bool, error) {
	defer s.reset()
	if s.err != nil {
		return false, s.err
	}
	query, args, err := s.builder.Limit(1).ToSql()
	if err != nil {
		return false, err
	}
	return s.engine.Exist(ctx, query, args...)
}

// Find returns all matching models.
func (s *QueryModelStmt[T]) Find(ctx context.Context) ([]T, error) {
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
	var t []T
	if err = ScanModels(rows, &t); err != nil {
		return nil, err
	}
	return t, nil
}

// Page returns the requested page of results together with the total row count.
func (s *QueryModelStmt[T]) Page(ctx context.Context, page, size uint64) ([]T, uint64, error) {
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
	offset := (page - 1) * size
	countStmt := QueryCol[uint64](s.engine)
	// Count on a cloned builder so filters stay in sync with the data query.
	countStmt.builder = s.builder.ToCountBuilder()
	count, ok, err := countStmt.Get(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !ok || count == 0 {
		return nil, 0, nil
	}
	if offset >= count {
		return nil, count, nil
	}
	s.builder.Limit(size).Offset(offset)
	list, err := s.Find(ctx)
	if err != nil {
		return nil, count, err
	}
	return list, count, nil
}

// Prefix adds an expression to the beginning of the query
func (s *QueryModelStmt[T]) Prefix(sql string, args ...any) *QueryModelStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *QueryModelStmt[T]) PrefixExpr(expr builder.Sqlizer) *QueryModelStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Distinct adds a DISTINCT clause to the query.
func (s *QueryModelStmt[T]) Distinct() *QueryModelStmt[T] {
	s.builder.Distinct()
	return s
}

// Options adds select option to the query
func (s *QueryModelStmt[T]) Options(options ...string) *QueryModelStmt[T] {
	s.builder.Options(options...)
	return s
}

// Select set result columns to the query.
func (s *QueryModelStmt[T]) Select(columns ...string) *QueryModelStmt[T] {
	s.builder.Select(columns...)
	return s
}

// AddColumn adds a result column to the query.
// Unlike Select, AddColumn accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *QueryModelStmt[T]) AddColumn(column any, args ...any) *QueryModelStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// RemoveColumns remove all columns from query.
// Must add a new column with Column or Select methods, otherwise
// return a error.
func (s *QueryModelStmt[T]) RemoveColumns() *QueryModelStmt[T] {
	s.builder.RemoveColumns()
	return s
}

// Column adds a result column to the query.
// Unlike Select, Column accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *QueryModelStmt[T]) Column(column any, args ...any) *QueryModelStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// From sets the FROM clause of the query.
func (s *QueryModelStmt[T]) From(from string) *QueryModelStmt[T] {
	s.builder.From(from)
	return s
}

// FromSelect sets a subquery into the FROM clause of the query.
func (s *QueryModelStmt[T]) FromSelect(from *builder.SelectBuilder, alias string) *QueryModelStmt[T] {
	s.builder.FromSelect(from, alias)
	return s
}

// JoinClause adds a join clause to the query.
func (s *QueryModelStmt[T]) JoinClause(pred any, args ...any) *QueryModelStmt[T] {
	s.builder.JoinClause(pred, args...)
	return s
}

// Join adds a JOIN clause to the query.
func (s *QueryModelStmt[T]) Join(join string, rest ...any) *QueryModelStmt[T] {
	s.builder.Join(join, rest...)
	return s
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (s *QueryModelStmt[T]) LeftJoin(join string, rest ...any) *QueryModelStmt[T] {
	s.builder.LeftJoin(join, rest...)
	return s
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (s *QueryModelStmt[T]) RightJoin(join string, rest ...any) *QueryModelStmt[T] {
	s.builder.RightJoin(join, rest...)
	return s
}

// InnerJoin adds a INNER JOIN clause to the query.
func (s *QueryModelStmt[T]) InnerJoin(join string, rest ...any) *QueryModelStmt[T] {
	s.builder.InnerJoin(join, rest...)
	return s
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (s *QueryModelStmt[T]) CrossJoin(join string, rest ...any) *QueryModelStmt[T] {
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
// bound to the placeholder. If the value is nil, the expression will be "<key>
// IS NULL". Slices and arrays are not expanded automatically; use builder.In or
// builder.NotIn explicitly when you need an IN-style predicate.
//
// Where will panic if pred isn't any of the above types.
func (s *QueryModelStmt[T]) Where(pred any, args ...any) *QueryModelStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate derived from the model metadata.
func (s *QueryModelStmt[T]) ID(id any) *QueryModelStmt[T] {
	var t T
	primaryKeys := t.LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		s.err = errors.New("lorm.Query().ID() only supports models with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{s.engine.Escaper().Escape(primaryKeys[0]): id})
	return s
}

// GroupBy adds GROUP BY expressions to the query.
func (s *QueryModelStmt[T]) GroupBy(groupBys ...string) *QueryModelStmt[T] {
	s.builder.GroupBy(groupBys...)
	return s
}

// Having adds an expression to the HAVING clause of the query.
//
// See Where.
func (s *QueryModelStmt[T]) Having(pred any, rest ...any) *QueryModelStmt[T] {
	s.builder.Having(escapePredicate(s.engine.Escaper(), pred), rest...)
	return s
}

// OrderByClause adds ORDER BY clause to the query.
func (s *QueryModelStmt[T]) OrderByClause(pred any, args ...any) *QueryModelStmt[T] {
	s.builder.OrderByClause(pred, args...)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *QueryModelStmt[T]) OrderBy(orderBys ...string) *QueryModelStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *QueryModelStmt[T]) Limit(limit uint64) *QueryModelStmt[T] {
	s.builder.Limit(limit)
	return s
}

// RemoveLimit removes the LIMIT clause.
func (s *QueryModelStmt[T]) RemoveLimit() *QueryModelStmt[T] {
	s.builder.RemoveLimit()
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *QueryModelStmt[T]) Offset(offset uint64) *QueryModelStmt[T] {
	s.builder.Offset(offset)
	return s
}

// RemoveOffset removes OFFSET clause.
func (s *QueryModelStmt[T]) RemoveOffset() *QueryModelStmt[T] {
	s.builder.RemoveOffset()
	return s
}

// Suffix adds an expression to the end of the query
func (s *QueryModelStmt[T]) Suffix(sql string, args ...any) *QueryModelStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *QueryModelStmt[T]) SuffixExpr(expr builder.Sqlizer) *QueryModelStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}

// QueryCol builds a SELECT statement that scans into a single column of T.
func QueryCol[T any](engine *Engine) *QueryColStmt[T] {
	return &QueryColStmt[T]{
		engine:  engine,
		builder: new(builder.SelectBuilder),
	}
}

// QueryColStmt is a fluent SELECT builder that scans scalar values of T.
type QueryColStmt[T any] struct {
	engine  *Engine
	builder *builder.SelectBuilder
}

func (s *QueryColStmt[T]) reset() {
	s.builder = new(builder.SelectBuilder)
}

// Get returns the first column value and whether a row was found.
func (s *QueryColStmt[T]) Get(ctx context.Context) (T, bool, error) {
	var t T
	defer s.reset()
	query, args, err := s.builder.ToSql()
	if err != nil {
		return t, false, err
	}
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		return t, false, err
	}
	defer rows.Close()
	if err = ScanCol(rows, &t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, false, nil
		}
		return t, false, err
	}
	return t, true, nil
}

// Find returns all values from the first selected column.
func (s *QueryColStmt[T]) Find(ctx context.Context) ([]T, error) {
	defer s.reset()
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	var t []T
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	if err = ScanCols(rows, &t); err != nil {
		return nil, err
	}
	return t, nil
}

// Prefix adds an expression to the beginning of the query
func (s *QueryColStmt[T]) Prefix(sql string, args ...any) *QueryColStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *QueryColStmt[T]) PrefixExpr(expr builder.Sqlizer) *QueryColStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Distinct adds a DISTINCT clause to the query.
func (s *QueryColStmt[T]) Distinct() *QueryColStmt[T] {
	s.builder.Distinct()
	return s
}

// Options adds select option to the query
func (s *QueryColStmt[T]) Options(options ...string) *QueryColStmt[T] {
	s.builder.Options(options...)
	return s
}

// Select set result columns to the query.
func (s *QueryColStmt[T]) Select(columns ...string) *QueryColStmt[T] {
	s.builder.Select(columns...)
	return s
}

// AddColumn adds a result column to the query.
// Unlike Select, AddColumn accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *QueryColStmt[T]) AddColumn(column any, args ...any) *QueryColStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// RemoveColumns remove all columns from query.
// Must add a new column with Column or Select methods, otherwise
// return a error.
func (s *QueryColStmt[T]) RemoveColumns() *QueryColStmt[T] {
	s.builder.RemoveColumns()
	return s
}

// Column adds a result column to the query.
// Unlike Select, Column accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *QueryColStmt[T]) Column(column any, args ...any) *QueryColStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// From sets the FROM clause of the query.
func (s *QueryColStmt[T]) From(from string) *QueryColStmt[T] {
	s.builder.From(from)
	return s
}

// FromSelect sets a subquery into the FROM clause of the query.
func (s *QueryColStmt[T]) FromSelect(from *builder.SelectBuilder, alias string) *QueryColStmt[T] {
	s.builder.FromSelect(from, alias)
	return s
}

// JoinClause adds a join clause to the query.
func (s *QueryColStmt[T]) JoinClause(pred any, args ...any) *QueryColStmt[T] {
	s.builder.JoinClause(pred, args...)
	return s
}

// Join adds a JOIN clause to the query.
func (s *QueryColStmt[T]) Join(join string, rest ...any) *QueryColStmt[T] {
	s.builder.Join(join, rest...)
	return s
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (s *QueryColStmt[T]) LeftJoin(join string, rest ...any) *QueryColStmt[T] {
	s.builder.LeftJoin(join, rest...)
	return s
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (s *QueryColStmt[T]) RightJoin(join string, rest ...any) *QueryColStmt[T] {
	s.builder.RightJoin(join, rest...)
	return s
}

// InnerJoin adds a INNER JOIN clause to the query.
func (s *QueryColStmt[T]) InnerJoin(join string, rest ...any) *QueryColStmt[T] {
	s.builder.InnerJoin(join, rest...)
	return s
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (s *QueryColStmt[T]) CrossJoin(join string, rest ...any) *QueryColStmt[T] {
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
// bound to the placeholder. If the value is nil, the expression will be "<key>
// IS NULL". Slices and arrays are not expanded automatically; use builder.In or
// builder.NotIn explicitly when you need an IN-style predicate.
//
// Where will panic if pred isn't any of the above types.
func (s *QueryColStmt[T]) Where(pred any, args ...any) *QueryColStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// GroupBy adds GROUP BY expressions to the query.
func (s *QueryColStmt[T]) GroupBy(groupBys ...string) *QueryColStmt[T] {
	s.builder.GroupBy(groupBys...)
	return s
}

// Having adds an expression to the HAVING clause of the query.
//
// See Where.
func (s *QueryColStmt[T]) Having(pred any, rest ...any) *QueryColStmt[T] {
	s.builder.Having(escapePredicate(s.engine.Escaper(), pred), rest...)
	return s
}

// OrderByClause adds ORDER BY clause to the query.
func (s *QueryColStmt[T]) OrderByClause(pred any, args ...any) *QueryColStmt[T] {
	s.builder.OrderByClause(pred, args...)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *QueryColStmt[T]) OrderBy(orderBys ...string) *QueryColStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *QueryColStmt[T]) Limit(limit uint64) *QueryColStmt[T] {
	s.builder.Limit(limit)
	return s
}

// RemoveLimit removes the LIMIT clause.
func (s *QueryColStmt[T]) RemoveLimit() *QueryColStmt[T] {
	s.builder.RemoveLimit()
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *QueryColStmt[T]) Offset(offset uint64) *QueryColStmt[T] {
	s.builder.Offset(offset)
	return s
}

// RemoveOffset removes OFFSET clause.
func (s *QueryColStmt[T]) RemoveOffset() *QueryColStmt[T] {
	s.builder.RemoveOffset()
	return s
}

// Suffix adds an expression to the end of the query
func (s *QueryColStmt[T]) Suffix(sql string, args ...any) *QueryColStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *QueryColStmt[T]) SuffixExpr(expr builder.Sqlizer) *QueryColStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
