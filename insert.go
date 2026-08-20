package lorm

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/yvvlee/lorm/builder"
)

func newInsertBuilder[T Table](engine *Engine) *builder.InsertBuilder {
	var t T
	return builder.Insert(engine.Escaper().Escape(t.TableName()))
}

func newInsertStmt[T Table](engine *Engine, isNil func(T) bool) *InsertStmt[T] {
	return &InsertStmt[T]{
		engine:  engine,
		builder: newInsertBuilder[T](engine),
		isNil:   isNil,
	}
}

// Insert builds an INSERT statement for pointer table P.
func (e *Engine) Insert[P TablePointer[M], M any]() *InsertStmt[P] {
	return newInsertStmt(e, func(model P) bool { return model == nil })
}

// InsertStmt is a fluent INSERT builder for table T.
type InsertStmt[T Table] struct {
	engine            *Engine
	builder           *builder.InsertBuilder
	models            []T
	isNil             func(T) bool
	err               error
	requireIDBackfill bool
}

func (s *InsertStmt[T]) reset() {
	s.builder = newInsertBuilder[T](s.engine)
	s.models = s.models[:0]
	s.err = nil
	s.requireIDBackfill = false
}

// Clone returns a copy of the statement state. Terminal methods still reset
// only the statement they are called on.
func (s *InsertStmt[T]) Clone() *InsertStmt[T] {
	return &InsertStmt[T]{
		engine:            s.engine,
		builder:           s.builder.Clone(),
		models:            append([]T(nil), s.models...),
		isNil:             s.isNil,
		err:               s.err,
		requireIDBackfill: s.requireIDBackfill,
	}
}

// AddModel appends a model to the insert batch.
func (s *InsertStmt[T]) AddModel(model T) *InsertStmt[T] {
	s.models = append(s.models, model)
	return s
}

// AddModels appends models to the insert batch.
func (s *InsertStmt[T]) AddModels(models ...T) *InsertStmt[T] {
	s.models = append(s.models, models...)
	return s
}

// Ignore enables the dialect's native conflict-ignore behavior.
func (s *InsertStmt[T]) Ignore() *InsertStmt[T] {
	switch s.engine.IgnoreStrategy() {
	case IgnoreConflictSuffix:
		s.builder.Suffix("ON CONFLICT DO NOTHING")
	case IgnoreOrKeyword:
		s.builder.StatementKeyword("INSERT OR IGNORE")
	default:
		s.builder.StatementKeyword("INSERT IGNORE")
	}
	return s
}

// RequireIDBackfill makes a multi-model insert execute one statement per model
// so every generated ID has an unambiguous destination.
func (s *InsertStmt[T]) RequireIDBackfill() *InsertStmt[T] {
	s.requireIDBackfill = true
	return s
}

// Exec executes the INSERT and backfills unambiguous generated primary keys.
func (s *InsertStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	defer s.reset()
	if s.err != nil {
		return 0, s.err
	}
	if len(s.models) == 0 {
		return 0, nil
	}
	for i, model := range s.models {
		if s.isNil(model) {
			return 0, fmt.Errorf("lorm.Insert().Exec() model at index %d is nil", i)
		}
	}

	now := time.Now()
	plans := make([]InsertPlan, len(s.models))
	var sharedColumns []string
	if _, hasHook := any(s.models[0]).(BeforeInsertHook); !hasHook {
		descriptor := s.models[0].LormModelDescriptor()
		if descriptor == nil {
			return 0, fmt.Errorf("lorm: prepare insert model at index 0: model %T returned a nil descriptor", s.models[0])
		}
		sharedColumns = descriptor.AllFields()
	}
	for i, model := range s.models {
		plans[i], err = prepareInsertPlan(model, now, sharedColumns)
		if err != nil {
			return 0, fmt.Errorf("lorm: prepare insert model at index %d: %w", i, err)
		}
		if err := validateInsertPlan(plans[i], i); err != nil {
			return 0, err
		}
	}

	if len(plans) > 1 && s.requireIDBackfill {
		if hasGeneratedInsertPlan(plans) && !s.engine.SupportsReturning() && !s.engine.SupportsLastInsertId() {
			return 0, fmt.Errorf("lorm: required ID backfill is not supported by driver %q", s.engine.DriverName())
		}
		return s.execOneByOne(ctx, plans)
	}
	if len(plans) == 1 {
		return s.execPlans(ctx, s.builder, s.models, plans, plans[0].AutoIncrementZero)
	}
	return s.execBatch(ctx, plans)
}

func validateInsertPlan(plan InsertPlan, index int) error {
	if len(plan.Columns) != len(plan.Values) {
		return fmt.Errorf(
			"lorm: insert plan at index %d has %d columns and %d values",
			index,
			len(plan.Columns),
			len(plan.Values),
		)
	}
	if plan.AutoIncrementZero && plan.AutoIncrementColumn == "" {
		return fmt.Errorf("lorm: insert plan at index %d marks an empty auto-increment column as zero", index)
	}
	return nil
}

func hasGeneratedInsertPlan(plans []InsertPlan) bool {
	for _, plan := range plans {
		if plan.AutoIncrementZero {
			return true
		}
	}
	return false
}

func sameInsertShape(left, right InsertPlan) bool {
	return left.AutoIncrementColumn == right.AutoIncrementColumn &&
		left.AutoIncrementZero == right.AutoIncrementZero &&
		sameInsertColumns(left.Columns, right.Columns)
}

func sameInsertColumns(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 || &left[0] == &right[0] {
		return true
	}
	return slices.Equal(left, right)
}

func (s *InsertStmt[T]) execBatch(ctx context.Context, plans []InsertPlan) (rowsAffected int64, err error) {
	firstGroupEnd := len(plans)
	for i := 1; i < len(plans); i++ {
		if !sameInsertShape(plans[0], plans[i]) {
			firstGroupEnd = i
			break
		}
	}
	if firstGroupEnd == len(plans) {
		return s.execPlans(ctx, s.builder, s.models, plans, false)
	}

	err = s.engine.TX(ctx, func(txCtx context.Context) error {
		for start := 0; start < len(plans); {
			end := start + 1
			for end < len(plans) && sameInsertShape(plans[start], plans[end]) {
				end++
			}
			rows, execErr := s.execPlans(
				txCtx,
				s.builder.Clone(),
				s.models[start:end],
				plans[start:end],
				false,
			)
			if execErr != nil {
				return execErr
			}
			rowsAffected += rows
			start = end
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (s *InsertStmt[T]) execOneByOne(ctx context.Context, plans []InsertPlan) (rowsAffected int64, err error) {
	err = s.engine.TX(ctx, func(txCtx context.Context) error {
		for i, model := range s.models {
			rows, execErr := s.execPlans(
				txCtx,
				s.builder.Clone(),
				[]T{model},
				plans[i:i+1],
				plans[i].AutoIncrementZero,
			)
			if execErr != nil {
				return execErr
			}
			rowsAffected += rows
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (s *InsertStmt[T]) execPlans(
	ctx context.Context,
	insertBuilder *builder.InsertBuilder,
	models []T,
	plans []InsertPlan,
	backfillID bool,
) (rowsAffected int64, err error) {
	afterHook, hasAfterHook := any(models[0]).(AfterInsertHook)
	if backfillID && hasAfterHook && s.engine.SupportsReturning() {
		return s.execWithReturning(ctx, insertBuilder, afterHook, plans[0])
	}

	result, err := s.execInsert(ctx, insertBuilder, plans)
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if backfillID && hasAfterHook && rowsAffected > 0 && s.engine.SupportsLastInsertId() {
		generatedID, idErr := result.LastInsertId()
		if idErr != nil {
			return 0, idErr
		}
		if err = afterHook.LormAfterInsert(InsertResult{
			RowsAffected:   rowsAffected,
			GeneratedID:    generatedID,
			HasGeneratedID: true,
		}); err != nil {
			return 0, err
		}
	}
	return rowsAffected, nil
}

func (s *InsertStmt[T]) execInsert(
	ctx context.Context,
	insertBuilder *builder.InsertBuilder,
	plans []InsertPlan,
) (sql.Result, error) {
	first := plans[0]
	escaper := s.engine.Escaper()
	columns := make([]string, len(first.Columns))
	for i, column := range first.Columns {
		columns[i] = escaper.Escape(column)
	}

	insertBuilder.Columns(columns...)
	for i, plan := range plans {
		if !sameInsertShape(first, plan) {
			return nil, fmt.Errorf("lorm: insert plan at index %d has a different column shape", i)
		}
		insertBuilder.Values(plan.Values...)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	return s.engine.Exec(ctx, query, args...)
}

func (s *InsertStmt[T]) execWithReturning(
	ctx context.Context,
	insertBuilder *builder.InsertBuilder,
	afterHook AfterInsertHook,
	plan InsertPlan,
) (rowsAffected int64, err error) {
	escaper := s.engine.Escaper()
	columns := make([]string, len(plan.Columns))
	for i, column := range plan.Columns {
		columns[i] = escaper.Escape(column)
	}
	insertBuilder.Columns(columns...).Values(plan.Values...).Returning(escaper.Escape(plan.AutoIncrementColumn))

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return 0, err
	}
	rows, err := s.engine.SQL(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	var generatedID int64
	if err = rows.Scan(&generatedID); err != nil {
		return 0, err
	}
	if rows.Next() {
		return 0, fmt.Errorf("lorm: single-row insert returned more than one generated ID")
	}
	if err = rows.Err(); err != nil {
		return 0, err
	}
	if err = afterHook.LormAfterInsert(InsertResult{
		RowsAffected:   1,
		GeneratedID:    generatedID,
		HasGeneratedID: true,
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

// columns overrides the INSERT column list for package-internal extensions.
func (s *InsertStmt[T]) columns(columns ...string) *InsertStmt[T] {
	s.builder.Columns(columns...)
	return s
}

// values appends a raw VALUES row to the underlying builder.
func (s *InsertStmt[T]) values(values ...any) *InsertStmt[T] {
	s.builder.Values(values...)
	return s
}

func (s *InsertStmt[T]) Prefix(sql string, args ...any) *InsertStmt[T] {
	s.builder.Prefix(sql, args...)
	return s
}

func (s *InsertStmt[T]) PrefixExpr(expr builder.Sqlizer) *InsertStmt[T] {
	s.builder.PrefixExpr(expr)
	return s
}

func (s *InsertStmt[T]) Suffix(sql string, args ...any) *InsertStmt[T] {
	s.builder.Suffix(sql, args...)
	return s
}

func (s *InsertStmt[T]) SuffixExpr(expr builder.Sqlizer) *InsertStmt[T] {
	s.builder.SuffixExpr(expr)
	return s
}
