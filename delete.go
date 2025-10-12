package lorm

import (
	"context"

	"github.com/yvvlee/lorm/builder"
)

func Delete(engine *Engine) *DeleteStmt {
	return &DeleteStmt{
		engine:  engine,
		builder: new(builder.DeleteBuilder),
	}
}

type DeleteStmt struct {
	engine  *Engine
	builder *builder.DeleteBuilder
}

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
	s.builder.Where(pred, args...)
	return s
}

func (s *DeleteStmt) ID(id any) *DeleteStmt {
	s.builder.Where("id = ?", id)
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
