package lorm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/samber/lo"
	"github.com/yvvlee/lorm/builder"
)

func Query[T Model](engine *Engine) *QueryModelStmt[T] {
	var t T
	fields := t.LormModelDescriptor().AllFields()
	if escaper := engine.Escaper(); escaper != nil {
		fields = lo.Map(fields, func(field string, _ int) string {
			return escaper.Escape(field)
		})
	}
	selectBuilder := builder.Select(fields...)
	if table, ok := any(t).(Table); ok {
		selectBuilder.From(table.TableName())
	}
	return &QueryModelStmt[T]{
		engine:  engine,
		builder: selectBuilder,
	}
}

type QueryModelStmt[T Model] struct {
	engine  *Engine
	builder *builder.SelectBuilder
}

func (s *QueryModelStmt[T]) Get(ctx context.Context) (T, error) {
	var t T
	query, args, err := s.builder.ToSql()
	if err != nil {
		return t, err
	}
	res := t.New()
	err = s.engine.Query(ctx, NewModelScanner(res), query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, nil
		}
		return t, err
	}
	return res.(T), nil
}

func (s *QueryModelStmt[T]) Exist(ctx context.Context) (bool, error) {
	query, args, err := s.builder.ToSql()
	if err != nil {
		return false, err
	}
	return s.engine.Exist(ctx, query, args...)
}

func (s *QueryModelStmt[T]) Find(ctx context.Context) ([]T, error) {
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	var t []T
	err = s.engine.Query(ctx, NewModelsScanner(&t), query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (s *QueryModelStmt[T]) Page(ctx context.Context, page, size uint64) ([]T, uint64, error) {
	if size == 0 {
		return nil, 0, errors.New("size can not be zero")
	}
	if page == 0 {
		page = 1
	}
	offset := (page - 1) * size
	s.builder.Limit(size).Offset(offset)
	countStmt := QueryCol[uint64](s.engine)
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
// IS NULL". If the value is an array or slice, the expression will be "<key> IN
// (?,?,...)", with one placeholder for each item in the value. These expressions
// are ANDed together.
//
// Where will panic if pred isn't any of the above types.
func (s *QueryModelStmt[T]) Where(pred any, args ...any) *QueryModelStmt[T] {
	s.builder.Where(pred, args...)
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
	s.builder.Having(pred, rest...)
	return s
}

// OrderByClause adds ORDER BY clause to the query.
func (s *QueryModelStmt[T]) OrderByClause(pred any, args ...any) *QueryModelStmt[T] {
	s.builder.Having(pred, args...)
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

func QueryCol[T any](engine *Engine) *QueryColStmt[T] {
	return &QueryColStmt[T]{
		engine:  engine,
		builder: new(builder.SelectBuilder),
	}
}

type QueryColStmt[T any] struct {
	engine  *Engine
	builder *builder.SelectBuilder
}

func (s *QueryColStmt[T]) Get(ctx context.Context) (T, bool, error) {
	var t T
	query, args, err := s.builder.ToSql()
	if err != nil {
		return t, false, err
	}
	err = s.engine.Query(ctx, NewColScanner(&t), query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, false, nil
		}
		return t, false, err
	}
	return t, true, nil
}

func (s *QueryColStmt[T]) Find(ctx context.Context) ([]T, error) {
	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	var t []T
	err = s.engine.Query(ctx, NewColsScanner(&t), query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
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

// Columns adds result columns to the query.
func (s *QueryColStmt[T]) Columns(columns ...string) *QueryColStmt[T] {
	s.builder.Select(columns...)
	return s
}

// RemoveColumns remove all columns from query.
// Must add a new column with Column or Columns methods, otherwise
// return a error.
func (s *QueryColStmt[T]) RemoveColumns() *QueryColStmt[T] {
	s.builder.RemoveColumns()
	return s
}

// Column adds a result column to the query.
// Unlike Columns, Column accepts args which will be bound to placeholders in
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
// IS NULL". If the value is an array or slice, the expression will be "<key> IN
// (?,?,...)", with one placeholder for each item in the value. These expressions
// are ANDed together.
//
// Where will panic if pred isn't any of the above types.
func (s *QueryColStmt[T]) Where(pred any, args ...any) *QueryColStmt[T] {
	s.builder.Where(pred, args...)
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
	s.builder.Having(pred, rest...)
	return s
}

// OrderByClause adds ORDER BY clause to the query.
func (s *QueryColStmt[T]) OrderByClause(pred any, args ...any) *QueryColStmt[T] {
	s.builder.Having(pred, args...)
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
