package lorm

import (
	"context"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/yvvlee/lorm/builder"
)

func Update(engine *Engine) *UpdateStmt {
	return &UpdateStmt{
		engine:  engine,
		builder: new(builder.UpdateBuilder),
	}
}

type UpdateStmt struct {
	engine  *Engine
	builder *builder.UpdateBuilder
}

func (s *UpdateStmt) Exec(ctx context.Context) (rowsAffected int64, err error) {
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

func (s *UpdateStmt) Table(table string) *UpdateStmt {
	s.builder.Table(table)
	return s
}

// Prefix adds an expression to the beginning of the query
func (s *UpdateStmt) Prefix(sql string, args ...any) *UpdateStmt {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *UpdateStmt) PrefixExpr(expr builder.Sqlizer) *UpdateStmt {
	s.builder.PrefixExpr(expr)
	return s
}

// Set adds SET clauses to the query.
func (s *UpdateStmt) Set(column string, value any) *UpdateStmt {
	s.builder.Set(column, value)
	return s
}

func (s *UpdateStmt) SetModel(model Model) *UpdateStmt {
	escaper := s.engine.Escaper()
	if t, ok := model.(Table); ok {
		s.builder.Table(escaper.Escape(t.TableName()))
	}
	fieldMap := model.LormFieldMap()
	descriptor := model.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey)
	if len(primaryKeys) > 0 {
		s.builder.Where(lo.PickByKeys(fieldMap, primaryKeys))
	}
	updatedFields := descriptor.FlagFields(FlagUpdated)
	jsonFields := descriptor.FlagFields(FlagJson)
	now := time.Now()
	dataMap := lo.MapEntries(fieldMap, func(key string, value any) (string, any) {
		if slices.Contains(updatedFields, key) {
			fillCurrentTime(value, now)
		}
		if slices.Contains(jsonFields, key) {
			value = NewJSONFieldWrapper(value)
		}
		return escaper.Escape(key), value
	})
	s.builder.SetMap(dataMap)
	return s
}

// SetMap is a convenience method which calls .Set for each key/value pair in clauses.
func (s *UpdateStmt) SetMap(clauses map[string]any) *UpdateStmt {
	s.builder.SetMap(clauses)
	return s
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (s *UpdateStmt) Where(pred any, args ...any) *UpdateStmt {
	s.builder.Where(pred, args...)
	return s
}

func (s *UpdateStmt) ID(id any) *UpdateStmt {
	s.builder.Where("id = ?", id)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *UpdateStmt) OrderBy(orderBys ...string) *UpdateStmt {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *UpdateStmt) Limit(limit uint64) *UpdateStmt {
	s.builder.Limit(limit)
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *UpdateStmt) Offset(offset uint64) *UpdateStmt {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query
func (s *UpdateStmt) Suffix(sql string, args ...any) *UpdateStmt {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *UpdateStmt) SuffixExpr(expr builder.Sqlizer) *UpdateStmt {
	s.builder.SuffixExpr(expr)
	return s
}
