package lorm

import (
	"context"
	"database/sql"
	"sync"

	"github.com/samber/lo"

	"github.com/yvvlee/lorm/builder"
)

func Insert[T Table](engine *Engine) *InsertStmt[T] {
	var t T
	b := insertBuilderPool.Get().(*builder.InsertBuilder)
	b.Into(engine.Escaper().Escape(t.TableName()))
	return &InsertStmt[T]{
		engine:  engine,
		builder: b,
		models:  make([]T, 0),
	}
}

type InsertStmt[T Table] struct {
	engine  *Engine
	builder *builder.InsertBuilder
	models  []T
}

func (s *InsertStmt[T]) AddModel(model T) *InsertStmt[T] {
	s.models = append(s.models, model)
	return s
}

func (s *InsertStmt[T]) AddModels(models ...T) *InsertStmt[T] {
	s.models = append(s.models, models...)
	return s
}

func (s *InsertStmt[T]) Ignore() *InsertStmt[T] {
	switch s.engine.DriverName() {
	case "postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach":
		s.builder.Suffix("ON CONFLICT DO NOTHING")
	case "sqlite", "sqlite3":
		s.builder.StatementKeyword("INSERT OR IGNORE")
	case "oci8", "ora", "oracle", "goracle", "godror":
		panic("INSERT IGNORE is not supported in Oracle")
	case "sqlserver", "mssql", "azuresql":
		panic("INSERT IGNORE is not supported in SQL Server")
	default:
		s.builder.StatementKeyword("INSERT IGNORE")
	}
	return s
}

func (s *InsertStmt[T]) Exec(ctx context.Context) (rowsAffected int64, err error) {
	defer func() {
		s.builder.Clear()
		insertBuilderPool.Put(s.builder)
	}()

	if len(s.models) == 0 {
		return 0, nil
	}

	table := s.models[0]
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	var useReturning bool
	var pkColumn string
	if len(primaryKeys) == 1 {
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
		return item.LormFieldMap()[pkColumn]
	})
	rows, err := s.engine.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	err = ScanCols(rows, &primaryPointers)
	if err != nil {
		return 0, err
	}
	return int64(len(s.models)), nil
}

func (s *InsertStmt[T]) Columns(columns ...string) *InsertStmt[T] {
	s.builder.Columns(columns...)
	return s
}

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

var insertBuilderPool = sync.Pool{
	New: func() any {
		return builder.Insert("")
	},
}
