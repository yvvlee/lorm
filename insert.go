package lorm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/samber/lo"

	"github.com/yvvlee/lorm/builder"
)

// Insert builds an INSERT statement for table T.
func Insert[T Table](engine *Engine) *InsertStmt[T] {
	var t T
	b := builder.Insert(engine.Escaper().Escape(t.TableName()))
	return &InsertStmt[T]{
		engine:  engine,
		builder: b,
		models:  make([]T, 0),
	}
}

// InsertStmt is a fluent INSERT builder for table T.
type InsertStmt[T Table] struct {
	engine  *Engine
	builder *builder.InsertBuilder
	models  []T
	err     error
	ignore  bool
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

// Ignore enables duplicate-conflict suppression for drivers that support it.
func (s *InsertStmt[T]) Ignore() *InsertStmt[T] {
	s.ignore = true
	switch s.engine.DriverName() {
	case "postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach":
		s.builder.Suffix("ON CONFLICT DO NOTHING")
	case "sqlite", "sqlite3":
		s.builder.StatementKeyword("INSERT OR IGNORE")
	default:
		s.builder.StatementKeyword("INSERT IGNORE")
	}
	return s
}

// Exec executes the INSERT and backfills generated primary keys when possible.
//
// Generated primary keys are only backfilled when the driver can return a stable
// one-to-one mapping between inserted rows and generated values. Batch inserts on
// LastInsertId-only dialects intentionally do not infer or synthesize per-row IDs.
func (s *InsertStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	if s.err != nil {
		return 0, s.err
	}

	if len(s.models) == 0 {
		return 0, nil
	}

	table := s.models[0]
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	var useReturning bool
	var pkColumn string
	if len(primaryKeys) == 1 {
		// RETURNING can hydrate models only when there is a single generated key.
		pkColumn = primaryKeys[0]
		useReturning = s.engine.SupportsReturning()
	}

	if useReturning {
		return s.execWithReturning(ctx, pkColumn)
	}

	result, err := s.execInsert(ctx, pkColumn)
	if err != nil {
		return 0, err
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if len(s.models) == 1 && s.engine.SupportsLastInsertId() {
		return rowsAffected, fillModelID(table, result)
	}
	return rowsAffected, nil
}

func (s *InsertStmt[T]) execInsert(ctx context.Context, pkColumn string) (sql.Result, error) {
	fields, values := ModelsToInsertData(s.models, pkColumn)
	escaper := s.engine.Escaper()
	columns := lo.Map(fields, func(field string, _ int) string {
		return escaper.Escape(field)
	})

	s.builder.Columns(columns...)
	for _, value := range values {
		s.builder.Values(value...)
	}

	query, args, err := s.builder.ToSql()
	if err != nil {
		return nil, err
	}
	return s.engine.Exec(ctx, query, args...)
}

func (s *InsertStmt[T]) execWithReturning(ctx context.Context, pkColumn string) (rowsAffected int64, err error) {
	fields, values := ModelsToInsertData(s.models, pkColumn)
	escaper := s.engine.Escaper()
	columns := lo.Map(fields, func(field string, _ int) string {
		return escaper.Escape(field)
	})

	s.builder.Columns(columns...).Returning(escaper.Escape(pkColumn))
	for _, value := range values {
		s.builder.Values(value...)
	}

	query, args, err := s.builder.ToSql()
	if err != nil {
		return 0, err
	}

	primaryPointers := lo.Map(s.models, func(item T, _ int) any {
		return item.LormFieldPtr(pkColumn)
	})
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	returnedIDs := make([]any, 0, len(s.models))
	for rows.Next() {
		var returnedID any
		if err = rows.Scan(&returnedID); err != nil {
			return 0, err
		}
		returnedIDs = append(returnedIDs, returnedID)
	}
	if err = rows.Err(); err != nil {
		return 0, err
	}
	rowsAffected = int64(len(returnedIDs))
	if rowsAffected == 0 {
		return 0, nil
	}
	if !s.ignore && rowsAffected != int64(len(s.models)) {
		return 0, fmt.Errorf("expected %d returned rows, got %d", len(s.models), rowsAffected)
	}
	// INSERT IGNORE-style statements may skip conflicted rows and return fewer IDs.
	if s.ignore && rowsAffected != int64(len(s.models)) {
		return rowsAffected, nil
	}
	for i, returnedID := range returnedIDs {
		if err = fillModelPrimaryKey(primaryPointers[i], returnedID); err != nil {
			return 0, err
		}
	}
	return rowsAffected, nil
}

// Columns overrides the INSERT column list.
//
// Prefer AddModel/AddModels for model-based inserts. When Columns/Values are used
// together with AddModel/AddModels, the model-derived values still take effect
// during Exec.
func (s *InsertStmt[T]) Columns(columns ...string) *InsertStmt[T] {
	s.builder.Columns(columns...)
	return s
}

// Values appends a raw VALUES row to the underlying builder.
//
// Prefer AddModel/AddModels for model-based inserts. When Columns/Values are used
// together with AddModel/AddModels, the model-derived values still take effect
// during Exec.
func (s *InsertStmt[T]) Values(values ...any) *InsertStmt[T] {
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
