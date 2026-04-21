package lorm

import (
	"context"
	"errors"

	"github.com/yvvlee/lorm/builder"
)

// Delete builds a DELETE statement without model metadata.
func Delete(engine *Engine) *DeleteStmt {
	return &DeleteStmt{
		engine:  engine,
		builder: builder.Delete(""),
	}
}

// DeleteModel builds a DELETE statement preconfigured for table T.
func DeleteModel[T Table](engine *Engine) *DeleteModelStmt[T] {
	var t T
	return &DeleteModelStmt[T]{
		engine:  engine,
		builder: builder.Delete(engine.Escaper().Escape(t.TableName())),
	}
}

// DeleteStmt is a fluent DELETE builder without model metadata.
type DeleteStmt struct {
	engine  *Engine
	builder *builder.DeleteBuilder
}

// DeleteModelStmt is a fluent DELETE builder that uses model metadata.
type DeleteModelStmt[T Table] struct {
	engine  *Engine
	builder *builder.DeleteBuilder
	err     error
}

// Exec executes the built DELETE statement.
func (s *DeleteStmt) Exec(ctx context.Context) (rowsAffected int64, err error) {
	query, args, err := s.builder.ToSql()
	if err != nil {
		return 0, err
	}
	result, err := s.engine.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// From sets the target table.
func (s *DeleteStmt) From(from string) *DeleteStmt {
	s.builder.From(from)
	return s
}

// Prefix adds an expression to the beginning of the query
func (s *DeleteStmt) Prefix(sql string, args ...any) *DeleteStmt {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *DeleteStmt) PrefixExpr(expr builder.Sqlizer) *DeleteStmt {
	s.builder.PrefixExpr(expr)
	return s
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (s *DeleteStmt) Where(pred any, args ...any) *DeleteStmt {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a predicate on a literal id column.
func (s *DeleteStmt) ID(id any) *DeleteStmt {
	s.builder.Where("id = ?", id)
	return s
}

// Exec executes the built DELETE statement.
func (s *DeleteModelStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	if s.err != nil {
		return 0, s.err
	}
	query, args, err := s.builder.ToSql()
	if err != nil {
		return 0, err
	}
	result, err := s.engine.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Prefix adds an expression to the beginning of the query.
func (s *DeleteModelStmt[T]) Prefix(sql string, args ...any) *DeleteModelStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query.
func (s *DeleteModelStmt[T]) PrefixExpr(expr builder.Sqlizer) *DeleteModelStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Where adds WHERE expressions to the query.
func (s *DeleteModelStmt[T]) Where(pred any, args ...any) *DeleteModelStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate using the model metadata.
func (s *DeleteModelStmt[T]) ID(id any) *DeleteModelStmt[T] {
	if s.err != nil {
		return s
	}
	var t T
	primaryKeys := t.LormModelDescriptor().FlagFields(FlagPrimaryKey)
	escaper := s.engine.Escaper()
	if len(primaryKeys) != 1 {
		s.err = errors.New("lorm.DeleteModel().ID() only supports tables with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{escaper.Escape(primaryKeys[0]): id})
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *DeleteModelStmt[T]) OrderBy(orderBys ...string) *DeleteModelStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *DeleteModelStmt[T]) Limit(limit uint64) *DeleteModelStmt[T] {
	s.builder.Limit(limit)
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *DeleteModelStmt[T]) Offset(offset uint64) *DeleteModelStmt[T] {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query.
func (s *DeleteModelStmt[T]) Suffix(sql string, args ...any) *DeleteModelStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query.
func (s *DeleteModelStmt[T]) SuffixExpr(expr builder.Sqlizer) *DeleteModelStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *DeleteStmt) OrderBy(orderBys ...string) *DeleteStmt {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *DeleteStmt) Limit(limit uint64) *DeleteStmt {
	s.builder.Limit(limit)
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *DeleteStmt) Offset(offset uint64) *DeleteStmt {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query
func (s *DeleteStmt) Suffix(sql string, args ...any) *DeleteStmt {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *DeleteStmt) SuffixExpr(expr builder.Sqlizer) *DeleteStmt {
	s.builder.SuffixExpr(expr)
	return s
}
