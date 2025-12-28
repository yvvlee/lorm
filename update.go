package lorm

import (
	"context"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/yvvlee/lorm/builder"
)

func Update[T Table](engine *Engine) *UpdateStmt[T] {
	var t T
	updateBuilder := new(builder.UpdateBuilder)
	updateBuilder.Table(engine.Escaper().Escape(t.TableName()))
	return &UpdateStmt[T]{
		engine:  engine,
		builder: updateBuilder,
	}
}

type UpdateStmt[T Table] struct {
	engine  *Engine
	builder *builder.UpdateBuilder
}

func (s *UpdateStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
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

func (s *UpdateStmt[T]) Table(table string) *UpdateStmt[T] {
	s.builder.Table(table)
	return s
}

// Prefix adds an expression to the beginning of the query
func (s *UpdateStmt[T]) Prefix(sql string, args ...any) *UpdateStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query
func (s *UpdateStmt[T]) PrefixExpr(expr builder.Sqlizer) *UpdateStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

// Set adds SET clauses to the query.
func (s *UpdateStmt[T]) Set(column string, value any) *UpdateStmt[T] {
	s.builder.Set(column, value)
	return s
}

func (s *UpdateStmt[T]) SetModel(t T) *UpdateStmt[T] {
	escaper := s.engine.Escaper()
	fieldMap := t.LormFieldMap()
	descriptor := t.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey)
	if len(primaryKeys) > 0 {
		//add primary key condition
		pickMap := lo.PickByKeys(fieldMap, primaryKeys)
		pickMapWithEscapedKey := lo.MapKeys(pickMap, func(_ any, key string) string {
			return escaper.Escape(key)
		})
		s.builder.Where(pickMapWithEscapedKey)
	}
	versionKeys := descriptor.FlagFields(FlagVersion)
	if len(versionKeys) > 0 {
		//add version condition
		pickMap := lo.PickByKeys(fieldMap, versionKeys)
		pickMapWithEscapedKey := lo.MapKeys(pickMap, func(_ any, key string) string {
			return escaper.Escape(key)
		})
		s.builder.Where(pickMapWithEscapedKey)
	}
	createdFields := descriptor.FlagFields(FlagCreated)
	updatedFields := descriptor.FlagFields(FlagUpdated)
	jsonFields := descriptor.FlagFields(FlagJson)
	now := time.Now()
	fieldMap = lo.OmitByKeys(fieldMap, append(primaryKeys, createdFields...))
	dataMap := lo.MapEntries(fieldMap, func(key string, value any) (string, any) {
		if slices.Contains(updatedFields, key) {
			fillCurrentTime(value, now)
		}
		if slices.Contains(jsonFields, key) {
			value = NewJSONFieldWrapper(value)
		}
		if slices.Contains(versionKeys, key) {
			value = builder.Expr(escaper.Escape(key) + "+1")
		}
		return escaper.Escape(key), value
	})
	s.builder.SetMap(dataMap)
	return s
}

// SetMap is a convenience method which calls .Set for each key/value pair in clauses.
func (s *UpdateStmt[T]) SetMap(clauses map[string]any) *UpdateStmt[T] {
	s.builder.SetMap(clauses)
	return s
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (s *UpdateStmt[T]) Where(pred any, args ...any) *UpdateStmt[T] {
	s.builder.Where(pred, args...)
	return s
}

func (s *UpdateStmt[T]) ID(id any) *UpdateStmt[T] {
	s.builder.Where("id = ?", id)
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *UpdateStmt[T]) OrderBy(orderBys ...string) *UpdateStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *UpdateStmt[T]) Limit(limit uint64) *UpdateStmt[T] {
	s.builder.Limit(limit)
	return s
}

// Offset sets a OFFSET clause on the query.
func (s *UpdateStmt[T]) Offset(offset uint64) *UpdateStmt[T] {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query
func (s *UpdateStmt[T]) Suffix(sql string, args ...any) *UpdateStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query
func (s *UpdateStmt[T]) SuffixExpr(expr builder.Sqlizer) *UpdateStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
