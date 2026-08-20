package lorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/yvvlee/lorm/builder"
)

type defaultSelectProjectionCacheEntry struct {
	once       sync.Once
	projection *builder.PreparedProjection
	err        error
}

func newSelectBuilder[T Model](engine *Engine) *builder.SelectBuilder {
	var model T
	selectBuilder := new(builder.SelectBuilder)
	if table, ok := any(model).(Table); ok {
		selectBuilder.From(engine.Escaper().Escape(table.TableName()))
	}
	return selectBuilder
}

func (e *Engine) defaultSelectProjection(descriptor *ModelDescriptor) (*builder.PreparedProjection, error) {
	if descriptor == nil {
		return nil, errors.New("lorm: model descriptor is nil")
	}
	value, ok := e.defaultSelectProjections.Load(descriptor)
	if !ok {
		candidate := new(defaultSelectProjectionCacheEntry)
		value, _ = e.defaultSelectProjections.LoadOrStore(descriptor, candidate)
	}
	entry := value.(*defaultSelectProjectionCacheEntry)
	entry.once.Do(func() {
		entry.projection, entry.err = e.buildDefaultSelectProjection(descriptor)
	})
	return entry.projection, entry.err
}

func (e *Engine) buildDefaultSelectProjection(descriptor *ModelDescriptor) (*builder.PreparedProjection, error) {
	if len(descriptor.Fields) == 0 {
		return nil, fmt.Errorf("lorm: model descriptor %q has no fields", descriptor.Name)
	}

	escaper := e.Escaper()
	columns := make([]string, len(descriptor.Fields))
	for i, field := range descriptor.Fields {
		if field == nil {
			return nil, fmt.Errorf("lorm: model descriptor %q has a nil field at index %d", descriptor.Name, i)
		}
		if field.DBField == "" {
			return nil, fmt.Errorf("lorm: model descriptor %q has an empty database field at index %d", descriptor.Name, i)
		}
		columns[i] = escaper.Escape(field.DBField)
	}
	return builder.NewPreparedProjection(strings.Join(columns, ", ")), nil
}

func newSelectStmt[T Model](engine *Engine) *SelectStmt[T] {
	return &SelectStmt[T]{
		engine:  engine,
		builder: newSelectBuilder[T](engine),
	}
}

// Query builds a SELECT statement that scans rows into model pointer P.
func (e *Engine) Query[P ModelPointer[M], M any]() *SelectStmt[P] {
	return newSelectStmt[P](e)
}

// SelectStmt is a fluent SELECT builder that scans rows into model values.
type SelectStmt[T Model] struct {
	engine  *Engine
	builder *builder.SelectBuilder
	err     error
}

func (s *SelectStmt[T]) reset() {
	s.builder = newSelectBuilder[T](s.engine)
	s.err = nil
}

// Clone returns a copy of the statement state. Terminal methods still reset
// only the statement they are called on.
func (s *SelectStmt[T]) Clone() *SelectStmt[T] {
	return &SelectStmt[T]{
		engine:  s.engine,
		builder: s.builder.Clone(),
		err:     s.err,
	}
}

func (s *SelectStmt[T]) ensureSelectColumns() (bool, error) {
	if len(s.builder.GetColumns()) > 0 {
		return false, nil
	}
	var model T
	projection, err := s.engine.defaultSelectProjection(model.LormModelDescriptor())
	if err != nil {
		return false, err
	}
	s.builder.SelectPrepared(projection)
	return true, nil
}

// Get returns the first matching value and whether a row was found.
func (s *SelectStmt[T]) Get(ctx context.Context) (T, bool, error) {
	var t T
	defer s.reset()
	if s.err != nil {
		return t, false, s.err
	}
	orderedScan, err := s.ensureSelectColumns()
	if err != nil {
		return t, false, err
	}
	query, args, err := s.builder.Clone().Limit(1).ToSql()
	if err != nil {
		return t, false, err
	}
	rows, err := s.engine.SQL(ctx, query, args...)
	if err != nil {
		return t, false, err
	}
	defer rows.Close()
	var res T
	if orderedScan {
		res, err = scanOrderedModelValue[T](rows)
	} else {
		res, err = scanModelValue[T](rows)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, false, nil
		}
		return t, false, err
	}
	return res, true, nil
}

// GetCol returns the first selected column and whether a row was found.
func (s *SelectStmt[M]) GetCol[T any](ctx context.Context) (T, bool, error) {
	var value T
	defer s.reset()
	if s.err != nil {
		return value, false, s.err
	}
	if _, err := s.ensureSelectColumns(); err != nil {
		return value, false, err
	}
	query, args, err := s.builder.Clone().Limit(1).ToSql()
	if err != nil {
		return value, false, err
	}
	rows, err := s.engine.SQL(ctx, query, args...)
	if err != nil {
		return value, false, err
	}
	defer rows.Close()
	if err = ScanCol(rows, &value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return value, false, nil
		}
		return value, false, err
	}
	return value, true, nil
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
	orderedScan, err := s.ensureSelectColumns()
	if err != nil {
		return nil, err
	}
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := s.engine.SQL(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var values []T
	if orderedScan {
		values, err = scanOrderedModelValues[T](rows)
	} else {
		values, err = scanModelValues[T](rows)
	}
	if err != nil {
		return nil, err
	}
	return values, nil
}

// FindCols returns the selected column from all matching rows.
func (s *SelectStmt[M]) FindCols[T any](ctx context.Context) ([]T, error) {
	defer s.reset()
	if s.err != nil {
		return nil, s.err
	}
	if _, err := s.ensureSelectColumns(); err != nil {
		return nil, err
	}
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := s.engine.SQL(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	return scanColumnValues[T](rows)
}

// Page returns the requested page of results together with the total row count.
func (s *SelectStmt[T]) Page(ctx context.Context, page, size uint64) ([]T, uint64, error) {
	defer s.reset()
	if s.err != nil {
		return nil, 0, s.err
	}
	orderedScan, err := s.ensureSelectColumns()
	if err != nil {
		return nil, 0, err
	}
	scan := scanModelValues[T]
	if orderedScan {
		scan = scanOrderedModelValues[T]
	}
	return pageSelectValues(ctx, s.engine, s.builder, page, size, scan)
}

// PageCols returns one selected column for the requested page and the total row count.
func (s *SelectStmt[M]) PageCols[T any](ctx context.Context, page, size uint64) ([]T, uint64, error) {
	defer s.reset()
	if s.err != nil {
		return nil, 0, s.err
	}
	if _, err := s.ensureSelectColumns(); err != nil {
		return nil, 0, err
	}
	return pageSelectValues(ctx, s.engine, s.builder, page, size, scanColumnValues[T])
}

func pageSelectValues[T any](
	ctx context.Context,
	engine *Engine,
	selectBuilder *builder.SelectBuilder,
	page, size uint64,
	scan func(*sql.Rows) ([]T, error),
) ([]T, uint64, error) {
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
	count, err := querySelectCount(ctx, engine, selectBuilder)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	if offsetOverflow || offset >= count {
		return nil, count, nil
	}
	selectBuilder.Limit(size).Offset(offset)
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, count, err
	}
	rows, err := engine.SQL(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, count, nil
		}
		return nil, count, err
	}
	defer rows.Close()
	list, err := scan(rows)
	if err != nil {
		return nil, count, err
	}
	return list, count, nil
}

func querySelectCount(ctx context.Context, engine *Engine, selectBuilder *builder.SelectBuilder) (uint64, error) {
	query, args, err := selectBuilder.ToCountBuilder().ToSql()
	if err != nil {
		return 0, err
	}
	rows, err := engine.SQL(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var count uint64
	if err = ScanCol(rows, &count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
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
//	AddColumn("IF(col IN ("+lorm.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (s *SelectStmt[T]) AddColumn(column any, args ...any) *SelectStmt[T] {
	s.builder.AddColumn(column, args...)
	return s
}

// RemoveColumns removes all configured columns. Model columns are restored at
// execution time if no new column is added.
func (s *SelectStmt[T]) RemoveColumns() *SelectStmt[T] {
	s.builder.RemoveColumns()
	return s
}

// Column adds a result column to the query.
// Unlike Select, Column accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+lorm.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
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
	var model T
	descriptor := model.LormModelDescriptor()
	if descriptor == nil {
		s.err = errors.New("lorm.Engine.Query().ID() model descriptor is nil")
		return s
	}
	if len(descriptor.PrimaryKeys) != 1 {
		s.err = errors.New("lorm.Engine.Query().ID() only supports models with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{s.engine.Escaper().Escape(descriptor.PrimaryKeys[0]): id})
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
