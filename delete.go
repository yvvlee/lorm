package lorm

import (
	"context"
	"errors"

	"github.com/yvvlee/lorm/builder"
)

// Delete builds a DELETE statement for table T.
func Delete[T Table](engine *Engine) *DeleteStmt[T] {
	var t T
	return &DeleteStmt[T]{
		engine:  engine,
		builder: builder.Delete(engine.Escaper().Escape(t.TableName())),
	}
}

// DeleteStmt is a fluent DELETE builder for table T.
type DeleteStmt[T Table] struct {
	engine  *Engine
	builder *builder.DeleteBuilder
	err     error
}

// Exec executes the built DELETE statement.
func (s *DeleteStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
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

// From overrides the target table name.
func (s *DeleteStmt[T]) From(from string) *DeleteStmt[T] {
	s.builder.From(from)
	return s
}

// Prefix adds an expression to the beginning of the query.
func (s *DeleteStmt[T]) Prefix(sql string, args ...any) *DeleteStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query.
func (s *DeleteStmt[T]) PrefixExpr(expr builder.Sqlizer) *DeleteStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Where adds WHERE expressions to the query.
func (s *DeleteStmt[T]) Where(pred any, args ...any) *DeleteStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate using the model metadata.
func (s *DeleteStmt[T]) ID(id any) *DeleteStmt[T] {
	if s.err != nil {
		return s
	}
	var t T
	primaryKeys := t.LormModelDescriptor().FlagFields(FlagPrimaryKey)
	escaper := s.engine.Escaper()
	if len(primaryKeys) != 1 {
		s.err = errors.New("lorm.Delete().ID() only supports tables with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{escaper.Escape(primaryKeys[0]): id})
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *DeleteStmt[T]) OrderBy(orderBys ...string) *DeleteStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *DeleteStmt[T]) Limit(limit uint64) *DeleteStmt[T] {
	s.builder.Limit(limit)
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *DeleteStmt[T]) Offset(offset uint64) *DeleteStmt[T] {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query.
func (s *DeleteStmt[T]) Suffix(sql string, args ...any) *DeleteStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query.
func (s *DeleteStmt[T]) SuffixExpr(expr builder.Sqlizer) *DeleteStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
