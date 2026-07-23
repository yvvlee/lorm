package lorm

import (
	"context"
	"errors"
	"time"

	"github.com/yvvlee/lorm/builder"
)

func newUpdateBuilder[T Table](engine *Engine) *builder.UpdateBuilder {
	var t T
	return builder.Update(engine.Escaper().Escape(t.TableName()))
}

// Update builds an UPDATE statement for table T.
func Update[T Table](engine *Engine) *UpdateStmt[T] {
	return &UpdateStmt[T]{
		engine:  engine,
		builder: newUpdateBuilder[T](engine),
	}
}

// UpdateStmt is a fluent UPDATE builder for table T.
type UpdateStmt[T Table] struct {
	engine  *Engine
	builder *builder.UpdateBuilder
	after   []func(rowsAffected int64)
	err     error
}

func (s *UpdateStmt[T]) reset() {
	s.builder = newUpdateBuilder[T](s.engine)
	s.after = s.after[:0]
	s.err = nil
}

// Clone returns a copy of the statement state. Terminal methods still reset
// only the statement they are called on.
func (s *UpdateStmt[T]) Clone() *UpdateStmt[T] {
	return &UpdateStmt[T]{
		engine:  s.engine,
		builder: s.builder.Clone(),
		after:   append([]func(rowsAffected int64){}, s.after...),
		err:     s.err,
	}
}

// Exec executes the built UPDATE statement.
func (s *UpdateStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	defer s.reset()
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
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}
	for _, fn := range s.after {
		fn(rowsAffected)
	}
	return rowsAffected, nil
}

// Table overrides the target table name.
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
	s.builder.Set(s.engine.Escaper().Escape(column), value)
	return s
}

// SetModel maps model fields into SET and WHERE clauses using descriptor metadata.
//
// SetModel performs a full-field update for regular columns. Zero values are not
// ignored automatically, so partial updates should prefer SetMap/Set.
func (s *UpdateStmt[T]) SetModel(t T) *UpdateStmt[T] {
	if s.err != nil {
		return s
	}
	escaper := s.engine.Escaper()
	descriptor := t.LormModelDescriptor()
	valueAccessor, hasValueAccessor := any(t).(ModelFieldValueAccessor)
	var (
		hasPrimaryKey bool
		now           = time.Now()
		dataMap       = make(map[string]any, len(descriptor.Fields))
	)

	for _, field := range descriptor.Fields {
		valuePtr := t.LormFieldPtr(field.DBField)
		if valuePtr == nil {
			continue
		}
		value := valuePtr
		if hasValueAccessor {
			value = valueAccessor.LormFieldValue(field.DBField)
		}

		if field.Flag.HasFlag(FlagPrimaryKey) {
			// Primary keys identify the row to update instead of becoming SET values.
			hasPrimaryKey = true
			s.builder.Where(builder.Eq{escaper.Escape(field.DBField): value})
			continue
		}
		if field.Flag.HasFlag(FlagVersion) {
			// Version fields participate in optimistic locking and auto-increment.
			s.builder.Where(builder.Eq{escaper.Escape(field.DBField): value})
			dataMap[escaper.Escape(field.DBField)] = builder.Expr(escaper.Escape(field.DBField) + "+1")
			s.after = append(s.after, func(rowsAffected int64) {
				if rowsAffected > 0 {
					incrementVersionValue(valuePtr)
				}
			})
			continue
		}
		if field.Flag.HasFlag(FlagCreated) {
			continue
		}
		if field.Flag.HasFlag(FlagUpdated) {
			if updatedValue, syncValue, ok := newUpdatedFieldValue(valuePtr, now); ok {
				dataMap[escaper.Escape(field.DBField)] = updatedValue
				s.after = append(s.after, func(rowsAffected int64) {
					if rowsAffected > 0 {
						syncValue()
					}
				})
				continue
			}
		}
		dataMap[escaper.Escape(field.DBField)] = value
	}

	if !hasPrimaryKey {
		s.err = errors.New("lorm.Update().SetModel() requires tables with at least one primary key")
		return s
	}
	s.builder.SetMap(dataMap)
	return s
}

// SetMap is a convenience method which calls .Set for each key/value pair in clauses.
func (s *UpdateStmt[T]) SetMap(clauses map[string]any) *UpdateStmt[T] {
	s.builder.SetMap(escapeMap(s.engine.Escaper(), clauses))
	return s
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (s *UpdateStmt[T]) Where(pred any, args ...any) *UpdateStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate derived from the model metadata.
func (s *UpdateStmt[T]) ID(id any) *UpdateStmt[T] {
	if s.err != nil {
		return s
	}
	var t T
	primaryKeys := t.LormModelDescriptor().FlagFields(FlagPrimaryKey)
	escaper := s.engine.Escaper()
	if len(primaryKeys) != 1 {
		s.err = errors.New("lorm.Update().ID() only supports tables with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{escaper.Escape(primaryKeys[0]): id})
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
