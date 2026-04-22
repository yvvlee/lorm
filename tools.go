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
		fillZeroValue(v, now)
	case **time.Time:
		fillZeroPointerValue(v, now)
	case *int64:
		fillZeroValue(v, now.Unix())
	case **int64:
		fillZeroPointerValue(v, now.Unix())
	case *uint64:
		fillZeroValue(v, uint64(now.Unix()))
	case **uint64:
		fillZeroPointerValue(v, uint64(now.Unix()))
	case *int32:
		fillZeroValue(v, int32(now.Unix()))
	case **int32:
		fillZeroPointerValue(v, int32(now.Unix()))
	case *uint32:
		fillZeroValue(v, uint32(now.Unix()))
	case **uint32:
		fillZeroPointerValue(v, uint32(now.Unix()))
	case *int:
		fillZeroValue(v, int(now.Unix()))
	case **int:
		fillZeroPointerValue(v, int(now.Unix()))
	case *string:
		fillZeroValue(v, now.Format(time.DateTime))
	case **string:
		fillZeroPointerValue(v, now.Format(time.DateTime))
	}
}

func newUpdatedFieldValue(value any, now time.Time) (updatedValue any, syncValue func(), ok bool) {
	switch v := value.(type) {
	case *time.Time:
		return newUpdatedValue(v, now)
	case **time.Time:
		return newUpdatedPointerValue(v, now)
	case *int64:
		return newUpdatedValue(v, now.Unix())
	case **int64:
		return newUpdatedPointerValue(v, now.Unix())
	case *uint64:
		return newUpdatedValue(v, uint64(now.Unix()))
	case **uint64:
		return newUpdatedPointerValue(v, uint64(now.Unix()))
	case *int32:
		return newUpdatedValue(v, int32(now.Unix()))
	case **int32:
		return newUpdatedPointerValue(v, int32(now.Unix()))
	case *uint32:
		return newUpdatedValue(v, uint32(now.Unix()))
	case **uint32:
		return newUpdatedPointerValue(v, uint32(now.Unix()))
	case *int:
		return newUpdatedValue(v, int(now.Unix()))
	case **int:
		return newUpdatedPointerValue(v, int(now.Unix()))
	case *string:
		return newUpdatedValue(v, now.Format(time.DateTime))
	case **string:
		return newUpdatedPointerValue(v, now.Format(time.DateTime))
	default:
		return nil, nil, false
	}
}

func incrementVersionValue(value any) {
	switch v := value.(type) {
	case *uint64:
		incrementValue(v)
	case **uint64:
		incrementPointerValue(v)
	case *int64:
		incrementValue(v)
	case **int64:
		incrementPointerValue(v)
	case *uint32:
		incrementValue(v)
	case **uint32:
		incrementPointerValue(v)
	case *int32:
		incrementValue(v)
	case **int32:
		incrementPointerValue(v)
	case *uint16:
		incrementValue(v)
	case **uint16:
		incrementPointerValue(v)
	case *int16:
		incrementValue(v)
	case **int16:
		incrementPointerValue(v)
	case *uint8:
		incrementValue(v)
	case **uint8:
		incrementPointerValue(v)
	case *int8:
		incrementValue(v)
	case **int8:
		incrementPointerValue(v)
	case *uint:
		incrementValue(v)
	case **uint:
		incrementPointerValue(v)
	case *int:
		incrementValue(v)
	case **int:
		incrementPointerValue(v)
	}
}

type versionNumber interface {
	~uint64 | ~int64 | ~uint32 | ~int32 | ~uint16 | ~int16 | ~uint8 | ~int8 | ~uint | ~int
}

func fillZeroValue[T comparable](value *T, current T) {
	var zero T
	if *value == zero {
		*value = current
	}
}

func fillZeroPointerValue[T comparable](value **T, current T) {
	if *value == nil {
		currentValue := current
		*value = &currentValue
		return
	}
	fillZeroValue(*value, current)
}

func newUpdatedValue[T any](value *T, updated T) (updatedValue any, syncValue func(), ok bool) {
	return &updated, func() { *value = updated }, true
}

func newUpdatedPointerValue[T any](value **T, updated T) (updatedValue any, syncValue func(), ok bool) {
	return &updated, func() {
		if *value == nil {
			*value = new(T)
		}
		**value = updated
	}, true
}

func incrementValue[T versionNumber](value *T) {
	(*value)++
}

func incrementPointerValue[T versionNumber](value **T) {
	if *value == nil {
		current := T(1)
		*value = &current
		return
	}
	(**value)++
}
