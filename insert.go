package lorm

import (
	"context"
	"database/sql"
	"slices"

	"github.com/samber/lo"
	"github.com/yvvlee/lorm/builder"
)

func Insert[T Table](ctx context.Context, engine *Engine, table T) (rowsAffected int64, err error) {
	return InsertAll(ctx, engine, []T{table})
}

func InsertAll[T Table](ctx context.Context, engine *Engine, models []T) (rowsAffected int64, err error) {
	if len(models) == 0 {
		return
	}
	table := models[0]
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey)
	// Check if we can use RETURNING or LastInsertId
	var useReturning bool
	var pkColumn string
	if len(primaryKeys) == 1 {
		flagAutoIncrementFields := descriptor.FlagFields(FlagAutoIncrement)
		if slices.Contains(flagAutoIncrementFields, primaryKeys[0]) {
			pkColumn = primaryKeys[0]
			useReturning = engine.SupportsReturning()
		}
	}
	if useReturning {
		return insertsWithReturning(ctx, engine, models, pkColumn)
	}
	result, err := inserts(ctx, engine, models, pkColumn)
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return
	}
	if len(models) > 1 {
		return rowsAffected, nil
	}
	return rowsAffected, fillModelID(table, result)
}

func inserts[T Table](ctx context.Context, engine *Engine, models []T, pkColumn string) (sql.Result, error) {
	fields, values := ModelsToInsertData(models, pkColumn)
	escaper := engine.Escaper()
	columns := lo.Map(fields, func(field string, _ int) string {
		return escaper.Escape(field)
	})
	insertBuilder := builder.Insert(models[0].TableName()).Columns(columns...)
	for _, value := range values {
		insertBuilder.Values(value...)
	}
	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	return engine.Exec(ctx, query, args...)
}

func insertsWithReturning[T Table](ctx context.Context, engine *Engine, models []T, pkColumn string) (rowsAffected int64, err error) {
	fields, values := ModelsToInsertData(models, pkColumn)
	escaper := engine.Escaper()
	columns := lo.Map(fields, func(field string, _ int) string {
		return escaper.Escape(field)
	})

	insertBuilder := builder.Insert(models[0].TableName()).
		Columns(columns...).
		Returning(escaper.Escape(pkColumn))
	for _, value := range values {
		insertBuilder.Values(value...)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return 0, err
	}

	// Execute query and scan the returned ID
	primaryPointers := lo.Map(models, func(item T, _ int) any {
		return item.LormFieldMap()[pkColumn]
	})
	err = engine.Query(ctx, NewColsScanner(&primaryPointers), query, args...)

	if err != nil {
		return 0, err
	}
	return int64(len(models)), nil
}
