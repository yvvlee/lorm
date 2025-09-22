package lorm

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cast"
	"github.com/yvvlee/lorm/builder"
)

func Insert[T Table](ctx context.Context, engine *Engine, table T) (rowsAffected int64, err error) {
	result, err := inserts(ctx, engine, []T{table})
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return
	}
	return rowsAffected, fillModelID(table, result)
}

func InsertAll[T Table](ctx context.Context, engine *Engine, models []T) (rowsAffected int64, err error) {
	if len(models) == 0 {
		return
	}
	if len(models) == 1 {
		return Insert(ctx, engine, models[0])
	}

	result, err := inserts(ctx, engine, models)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func inserts[T Table](ctx context.Context, engine *Engine, models []T) (sql.Result, error) {
	table := models[0].TableName()
	insertBuilder := builder.Insert(table)
	fields, values := ModelsToInsertData(models)
	if escaper := engine.Escaper(); escaper != nil {
		insertBuilder.Into(escaper.Escape(table))
		insertBuilder.Columns(lo.Map(fields, func(field string, _ int) string {
			return escaper.Escape(field)
		})...)
	} else {
		insertBuilder.Columns(fields...)
	}
	for _, value := range values {
		insertBuilder.Values(value...)
	}
	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	return engine.DB(ctx).Exec(ctx, query, args...)
}

func fillCurrentTime(value any, now time.Time) {
	switch v := value.(type) {
	case *time.Time:
		if v.IsZero() {
			*v = now
		}
	case *int64:
		if *v == 0 {
			*v = now.Unix()
		}
	case *uint64:
		if *v == 0 {
			*v = uint64(now.Unix())
		}
	case *int32:
		if *v == 0 {
			*v = int32(now.Unix())
		}
	case *uint32:
		if *v == 0 {
			*v = uint32(now.Unix())
		}
	case *int:
		if *v == 0 {
			*v = int(now.Unix())
		}
	case *string:
		if *v == "" {
			*v = now.Format(time.DateTime)
		}
	}
}

func fillModelID(table Table, result sql.Result) error {
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return nil
	}
	flagAutoIncrementFields := descriptor.FlagFields(FlagAutoIncrement)
	if !slices.Contains(flagAutoIncrementFields, primaryKeys[0]) {
		return nil
	}
	primaryPointer := table.LormFieldMap()[primaryKeys[0]]
	if cast.ToUint64(primaryPointer) == 0 {
		lastInsertId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		switch id := primaryPointer.(type) {
		case *uint64:
			*id = cast.ToUint64(lastInsertId)
		case *int64:
			*id = cast.ToInt64(lastInsertId)
		case *uint32:
			*id = cast.ToUint32(lastInsertId)
		case *int32:
			*id = cast.ToInt32(lastInsertId)
		case *uint16:
			*id = cast.ToUint16(lastInsertId)
		case *int16:
			*id = cast.ToInt16(lastInsertId)
		case *uint8:
			*id = cast.ToUint8(lastInsertId)
		case *int8:
			*id = cast.ToInt8(lastInsertId)
		case *uint:
			*id = cast.ToUint(lastInsertId)
		case *int:
			*id = cast.ToInt(lastInsertId)
		}
	}
	return nil
}
