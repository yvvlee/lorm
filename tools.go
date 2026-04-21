package lorm

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spf13/cast"
)

func fillModelID(table Table, result sql.Result) error {
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	if len(primaryKeys) != 1 {
		return nil
	}
	primaryPointer := table.LormFieldPtr(primaryKeys[0])
	// Only backfill auto-generated keys when the model has not already set one.
	if cast.ToUint64(primaryPointer) == 0 {
		lastInsertId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		return fillModelPrimaryKey(primaryPointer, lastInsertId)
	}
	return nil
}

func fillModelPrimaryKey(primaryPointer any, value any) error {
	switch id := primaryPointer.(type) {
	case *uint64:
		*id = cast.ToUint64(value)
	case *int64:
		*id = cast.ToInt64(value)
	case *uint32:
		*id = cast.ToUint32(value)
	case *int32:
		*id = cast.ToInt32(value)
	case *uint16:
		*id = cast.ToUint16(value)
	case *int16:
		*id = cast.ToInt16(value)
	case *uint8:
		*id = cast.ToUint8(value)
	case *int8:
		*id = cast.ToInt8(value)
	case *uint:
		*id = cast.ToUint(value)
	case *int:
		*id = cast.ToInt(value)
	default:
		return fmt.Errorf("unsupported primary key type %T", primaryPointer)
	}
	return nil
}

func fillCurrentTime(value any, now time.Time) {
	// Populate managed time fields without overwriting explicit values.
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

func newUpdatedFieldValue(value any, now time.Time) (updatedValue any, syncValue func(), ok bool) {
	switch v := value.(type) {
	case *time.Time:
		updated := now
		return &updated, func() { *v = updated }, true
	case *int64:
		updated := now.Unix()
		return &updated, func() { *v = updated }, true
	case *uint64:
		updated := uint64(now.Unix())
		return &updated, func() { *v = updated }, true
	case *int32:
		updated := int32(now.Unix())
		return &updated, func() { *v = updated }, true
	case *uint32:
		updated := uint32(now.Unix())
		return &updated, func() { *v = updated }, true
	case *int:
		updated := int(now.Unix())
		return &updated, func() { *v = updated }, true
	case *string:
		updated := now.Format(time.DateTime)
		return &updated, func() { *v = updated }, true
	default:
		return nil, nil, false
	}
}

func incrementVersionValue(value any) {
	switch v := value.(type) {
	case *uint64:
		*v++
	case *int64:
		*v++
	case *uint32:
		*v++
	case *int32:
		*v++
	case *uint16:
		*v++
	case *int16:
		*v++
	case *uint8:
		*v++
	case *int8:
		*v++
	case *uint:
		*v++
	case *int:
		*v++
	}
}
