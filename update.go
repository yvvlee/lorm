package lorm

import (
	"context"
	"errors"
	"time"

	"github.com/yvvlee/lorm/builder"
)

type updateMode uint8

const (
	updateModeUnset updateMode = iota
	updateModeManual
	updateModeModel
)

func newUpdateBuilder[T Table](engine *Engine) *builder.UpdateBuilder {
	var t T
	return builder.Update(engine.Escaper().Escape(t.TableName()))
}

func newUpdateStmt[T Table](engine *Engine) *UpdateStmt[T] {
	return &UpdateStmt[T]{
		engine:  engine,
		builder: newUpdateBuilder[T](engine),
	}
}

// Update builds an UPDATE statement for pointer table P.
func (e *Engine) Update[P TablePointer[M], M any]() *UpdateStmt[P] {
	return newUpdateStmt[P](e)
}

// UpdateStmt is a fluent UPDATE builder for table T.
type UpdateStmt[T Table] struct {
	engine           *Engine
	builder          *builder.UpdateBuilder
	model            T
	modelNow         time.Time
	mode             updateMode
	allowGlobalWrite bool
	err              error
}

func (s *UpdateStmt[T]) reset() {
	s.builder = newUpdateBuilder[T](s.engine)
	var zero T
	s.model = zero
	s.modelNow = time.Time{}
	s.mode = updateModeUnset
	s.allowGlobalWrite = false
	s.err = nil
}

// Clone returns a copy of the statement state. Terminal methods still reset
// only the statement they are called on.
func (s *UpdateStmt[T]) Clone() *UpdateStmt[T] {
	return &UpdateStmt[T]{
		engine:           s.engine,
		builder:          s.builder.Clone(),
		model:            s.model,
		modelNow:         s.modelNow,
		mode:             s.mode,
		allowGlobalWrite: s.allowGlobalWrite,
		err:              s.err,
	}
}

// ToSql returns the query and arguments using the engine's placeholder format.
// It does not execute or reset the statement. Exec still checks write guards.
func (s *UpdateStmt[T]) ToSql() (string, []any, error) {
	if s.err != nil {
		return "", nil, s.err
	}
	return statementSQL(s.engine, s.builder)
}

// Exec executes the built UPDATE statement.
func (s *UpdateStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	defer s.reset()
	if s.err != nil {
		return 0, s.err
	}
	query, args, hasWhere, err := s.builder.ToSqlWithWhere()
	if err != nil {
		return 0, err
	}
	if !s.allowGlobalWrite && !hasWhere {
		return 0, errors.New("lorm.Update().Exec() requires a WHERE clause or AllowGlobalWrite()")
	}
	result, err := s.engine.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if s.mode == updateModeModel {
		if hook, ok := any(s.model).(AfterUpdateHook); ok {
			hook.LormAfterUpdate(s.modelNow, rowsAffected)
		}
	}
	return rowsAffected, nil
}

// Table overrides the target table name.
func (s *UpdateStmt[T]) Table(table string) *UpdateStmt[T] {
	s.builder.Table(table)
	return s
}

// Prefix adds an expression to the beginning of the query.
func (s *UpdateStmt[T]) Prefix(sql string, args ...any) *UpdateStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

// PrefixExpr adds an expression to the very beginning of the query.
func (s *UpdateStmt[T]) PrefixExpr(expr builder.Sqlizer) *UpdateStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

func (s *UpdateStmt[T]) enterManualMode() bool {
	if s.err != nil {
		return false
	}
	switch s.mode {
	case updateModeUnset:
		s.mode = updateModeManual
	case updateModeModel:
		s.err = errors.New("lorm.Update(): SetModel cannot be mixed with Set or SetMap")
		return false
	}
	return true
}

// Set adds a SET clause to the query.
func (s *UpdateStmt[T]) Set(column string, value any) *UpdateStmt[T] {
	if !s.enterManualMode() {
		return s
	}
	s.builder.Set(s.engine.Escaper().Escape(column), value)
	return s
}

// SetModel creates a full-field update plan for one model. The model must not be nil.
func (s *UpdateStmt[T]) SetModel(model T) *UpdateStmt[T] {
	if s.err != nil {
		return s
	}
	switch s.mode {
	case updateModeManual:
		s.err = errors.New("lorm.Update(): SetModel cannot be mixed with Set or SetMap")
		return s
	case updateModeModel:
		s.err = errors.New("lorm.Update().SetModel() can only be called once")
		return s
	}
	s.mode = updateModeModel
	s.model = model

	s.modelNow = time.Now()
	plan, err := prepareUpdatePlan(model, s.modelNow)
	if err != nil {
		s.err = err
		return s
	}
	if plan.PrimaryKeyCount == 0 {
		s.err = errors.New("lorm.Update().SetModel() requires tables with at least one primary key")
		return s
	}

	escaper := s.engine.Escaper()
	for _, item := range plan.Set {
		s.builder.Set(escaper.Escape(item.Column), item.Value)
	}
	for _, column := range plan.Increment {
		escaped := escaper.Escape(column)
		s.builder.Set(escaped, builder.Expr(escaped+"+1"))
	}
	for _, item := range plan.Where {
		s.builder.Where(builder.Eq{escaper.Escape(item.Column): item.Value})
	}
	return s
}

// AllowGlobalWrite explicitly permits an UPDATE without a restrictive WHERE clause.
func (s *UpdateStmt[T]) AllowGlobalWrite() *UpdateStmt[T] {
	s.allowGlobalWrite = true
	return s
}

// SetMap appends sorted SET clauses from clauses.
func (s *UpdateStmt[T]) SetMap(clauses map[string]any) *UpdateStmt[T] {
	if !s.enterManualMode() {
		return s
	}
	s.builder.SetMap(escapeMap(s.engine.Escaper(), clauses))
	return s
}

// Where adds WHERE expressions to the query.
func (s *UpdateStmt[T]) Where(pred any, args ...any) *UpdateStmt[T] {
	s.builder.Where(escapePredicate(s.engine.Escaper(), pred), args...)
	return s
}

// ID adds a single-column primary key predicate derived from model metadata.
func (s *UpdateStmt[T]) ID(id any) *UpdateStmt[T] {
	if s.err != nil {
		return s
	}
	var t T
	descriptor := t.LormModelDescriptor()
	if descriptor == nil {
		s.err = errors.New("lorm.Update().ID() model descriptor is nil")
		return s
	}
	escaper := s.engine.Escaper()
	if len(descriptor.PrimaryKeys) != 1 {
		s.err = errors.New("lorm.Update().ID() only supports tables with single-column primary keys")
		return s
	}
	s.builder.Where(builder.Eq{escaper.Escape(descriptor.PrimaryKeys[0]): id})
	return s
}

// OrderBy adds ORDER BY expressions to the query.
func (s *UpdateStmt[T]) OrderBy(orderBys ...string) *UpdateStmt[T] {
	s.builder.OrderBy(orderBys...)
	return s
}

// Asc appends columns in ascending order, quoting each column identifier.
// Columns must be names, not SQL expressions. Use OrderBy for expressions.
func (s *UpdateStmt[T]) Asc(columns ...string) *UpdateStmt[T] {
	for _, column := range columns {
		s.builder.OrderBy(s.engine.Escaper().Escape(column) + " ASC")
	}
	return s
}

// Desc appends columns in descending order, quoting each column identifier.
// Columns must be names, not SQL expressions. Use OrderBy for expressions.
func (s *UpdateStmt[T]) Desc(columns ...string) *UpdateStmt[T] {
	for _, column := range columns {
		s.builder.OrderBy(s.engine.Escaper().Escape(column) + " DESC")
	}
	return s
}

// Limit sets a LIMIT clause on the query.
func (s *UpdateStmt[T]) Limit(limit uint64) *UpdateStmt[T] {
	s.builder.Limit(limit)
	return s
}

// Offset sets an OFFSET clause on the query.
func (s *UpdateStmt[T]) Offset(offset uint64) *UpdateStmt[T] {
	s.builder.Offset(offset)
	return s
}

// Suffix adds an expression to the end of the query.
func (s *UpdateStmt[T]) Suffix(sql string, args ...any) *UpdateStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

// SuffixExpr adds an expression to the end of the query.
func (s *UpdateStmt[T]) SuffixExpr(expr builder.Sqlizer) *UpdateStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
