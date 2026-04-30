package lorm

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func fillModelID(table Table, result sql.Result) error {
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	if len(primaryKeys) != 1 {
		return nil
	}
	ptr := table.LormFieldPtr(primaryKeys[0])
	// Only backfill auto-generated keys when the model has not already set one.
	if primaryKeyIsZero(ptr) {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		return fillModelPrimaryKey(ptr, id)
	}
	return nil
}

func primaryKeyIsZero(ptr any) bool {
	switch p := ptr.(type) {
	case *int:
		return p == nil || *p == 0
	case *int8:
		return p == nil || *p == 0
	case *int16:
		return p == nil || *p == 0
	case *int32:
		return p == nil || *p == 0
	case *int64:
		return p == nil || *p == 0
	case *uint:
		return p == nil || *p == 0
	case *uint8:
		return p == nil || *p == 0
	case *uint16:
		return p == nil || *p == 0
	case *uint32:
		return p == nil || *p == 0
	case *uint64:
		return p == nil || *p == 0
	case **int:
		return p == nil || *p == nil || **p == 0
	case **int8:
		return p == nil || *p == nil || **p == 0
	case **int16:
		return p == nil || *p == nil || **p == 0
	case **int32:
		return p == nil || *p == nil || **p == 0
	case **int64:
		return p == nil || *p == nil || **p == 0
	case **uint:
		return p == nil || *p == nil || **p == 0
	case **uint8:
		return p == nil || *p == nil || **p == 0
	case **uint16:
		return p == nil || *p == nil || **p == 0
	case **uint32:
		return p == nil || *p == nil || **p == 0
	case **uint64:
		return p == nil || *p == nil || **p == 0
	default:
		return true
	}
}

func fillModelPrimaryKey(ptr any, value int64) error {
	switch p := ptr.(type) {
	case *int:
		if err := checkSignedPrimaryKey(ptr, value, strconv.IntSize); err != nil {
			return err
		}
		*p = int(value)
	case *int8:
		if err := checkSignedPrimaryKey(ptr, value, 8); err != nil {
			return err
		}
		*p = int8(value)
	case *int16:
		if err := checkSignedPrimaryKey(ptr, value, 16); err != nil {
			return err
		}
		*p = int16(value)
	case *int32:
		if err := checkSignedPrimaryKey(ptr, value, 32); err != nil {
			return err
		}
		*p = int32(value)
	case *int64:
		*p = value
	case *uint:
		if err := checkUnsignedPrimaryKey(ptr, value, strconv.IntSize); err != nil {
			return err
		}
		*p = uint(value)
	case *uint8:
		if err := checkUnsignedPrimaryKey(ptr, value, 8); err != nil {
			return err
		}
		*p = uint8(value)
	case *uint16:
		if err := checkUnsignedPrimaryKey(ptr, value, 16); err != nil {
			return err
		}
		*p = uint16(value)
	case *uint32:
		if err := checkUnsignedPrimaryKey(ptr, value, 32); err != nil {
			return err
		}
		*p = uint32(value)
	case *uint64:
		if err := checkUnsignedPrimaryKey(ptr, value, 64); err != nil {
			return err
		}
		*p = uint64(value)
	case **int:
		if err := checkSignedPrimaryKey(ptr, value, strconv.IntSize); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, int(value))
	case **int8:
		if err := checkSignedPrimaryKey(ptr, value, 8); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, int8(value))
	case **int16:
		if err := checkSignedPrimaryKey(ptr, value, 16); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, int16(value))
	case **int32:
		if err := checkSignedPrimaryKey(ptr, value, 32); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, int32(value))
	case **int64:
		return setPrimaryKeyPtr(p, value)
	case **uint:
		if err := checkUnsignedPrimaryKey(ptr, value, strconv.IntSize); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, uint(value))
	case **uint8:
		if err := checkUnsignedPrimaryKey(ptr, value, 8); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, uint8(value))
	case **uint16:
		if err := checkUnsignedPrimaryKey(ptr, value, 16); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, uint16(value))
	case **uint32:
		if err := checkUnsignedPrimaryKey(ptr, value, 32); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, uint32(value))
	case **uint64:
		if err := checkUnsignedPrimaryKey(ptr, value, 64); err != nil {
			return err
		}
		return setPrimaryKeyPtr(p, uint64(value))
	default:
		return fmt.Errorf("unsupported primary key type %T", ptr)
	}
	return nil
}

func checkSignedPrimaryKey(ptr any, value int64, bits int) error {
	if bits >= 64 {
		return nil
	}
	min := -int64(1 << (bits - 1))
	max := int64(1<<(bits-1)) - 1
	if value < min || value > max {
		return fmt.Errorf("primary key value %d overflows %T", value, ptr)
	}
	return nil
}

func checkUnsignedPrimaryKey(ptr any, value int64, bits int) error {
	if value < 0 {
		return fmt.Errorf("primary key value %d overflows %T", value, ptr)
	}
	if bits < 64 && uint64(value) > (uint64(1)<<bits)-1 {
		return fmt.Errorf("primary key value %d overflows %T", value, ptr)
	}
	return nil
}

func setPrimaryKeyPtr[T any](p **T, value T) error {
	if p == nil {
		return fmt.Errorf("unsupported primary key type %T", p)
	}
	if *p == nil {
		v := value
		*p = &v
		return nil
	}
	**p = value
	return nil
}

// fillCurrentTime populates managed time fields without overwriting explicit values.
func fillCurrentTime(value any, now time.Time) {
	switch v := value.(type) {
	case *time.Time:
		fillZero(v, now)
	case **time.Time:
		fillZeroPtr(v, now)
	case *sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			*v = sql.NullTime{Time: now, Valid: true}
		}
	case **sql.NullTime:
		fillZeroPtr(v, sql.NullTime{Time: now, Valid: true})
	case *int64:
		fillZero(v, now.Unix())
	case **int64:
		fillZeroPtr(v, now.Unix())
	case *uint64:
		fillZero(v, uint64(now.Unix()))
	case **uint64:
		fillZeroPtr(v, uint64(now.Unix()))
	case *int32:
		fillZero(v, int32(now.Unix()))
	case **int32:
		fillZeroPtr(v, int32(now.Unix()))
	case *uint32:
		fillZero(v, uint32(now.Unix()))
	case **uint32:
		fillZeroPtr(v, uint32(now.Unix()))
	case *int:
		fillZero(v, int(now.Unix()))
	case **int:
		fillZeroPtr(v, int(now.Unix()))
	case *string:
		fillZero(v, now.Format(time.DateTime))
	case **string:
		fillZeroPtr(v, now.Format(time.DateTime))
	}
}

func newUpdatedFieldValue(value any, now time.Time) (updatedValue any, syncValue func(), ok bool) {
	switch v := value.(type) {
	case *time.Time:
		return setUpdated(v, now)
	case **time.Time:
		return setUpdatedPtr(v, now)
	case *sql.NullTime:
		return setUpdated(v, sql.NullTime{Time: now, Valid: true})
	case **sql.NullTime:
		return setUpdatedPtr(v, sql.NullTime{Time: now, Valid: true})
	case *int64:
		return setUpdated(v, now.Unix())
	case **int64:
		return setUpdatedPtr(v, now.Unix())
	case *uint64:
		return setUpdated(v, uint64(now.Unix()))
	case **uint64:
		return setUpdatedPtr(v, uint64(now.Unix()))
	case *int32:
		return setUpdated(v, int32(now.Unix()))
	case **int32:
		return setUpdatedPtr(v, int32(now.Unix()))
	case *uint32:
		return setUpdated(v, uint32(now.Unix()))
	case **uint32:
		return setUpdatedPtr(v, uint32(now.Unix()))
	case *int:
		return setUpdated(v, int(now.Unix()))
	case **int:
		return setUpdatedPtr(v, int(now.Unix()))
	case *string:
		return setUpdated(v, now.Format(time.DateTime))
	case **string:
		return setUpdatedPtr(v, now.Format(time.DateTime))
	default:
		return nil, nil, false
	}
}

func incrementVersionValue(value any) {
	switch v := value.(type) {
	case *int:
		*v++
	case *int8:
		*v++
	case *int16:
		*v++
	case *int32:
		*v++
	case *int64:
		*v++
	case *uint:
		*v++
	case *uint8:
		*v++
	case *uint16:
		*v++
	case *uint32:
		*v++
	case *uint64:
		*v++
	case **int:
		incrPtr(v)
	case **int8:
		incrPtr(v)
	case **int16:
		incrPtr(v)
	case **int32:
		incrPtr(v)
	case **int64:
		incrPtr(v)
	case **uint:
		incrPtr(v)
	case **uint8:
		incrPtr(v)
	case **uint16:
		incrPtr(v)
	case **uint32:
		incrPtr(v)
	case **uint64:
		incrPtr(v)
	}
}

// --- generic helpers ---

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func fillZero[T comparable](p *T, val T) {
	var zero T
	if *p == zero {
		*p = val
	}
}

func fillZeroPtr[T comparable](p **T, val T) {
	if *p == nil {
		v := val
		*p = &v
	} else {
		fillZero(*p, val)
	}
}

func setUpdated[T any](p *T, val T) (any, func(), bool) {
	return &val, func() { *p = val }, true
}

func setUpdatedPtr[T any](p **T, val T) (any, func(), bool) {
	return &val, func() {
		if *p == nil {
			*p = new(T)
		}
		**p = val
	}, true
}

func incrPtr[T integer](p **T) {
	if *p == nil {
		v := T(1)
		*p = &v
	} else {
		**p++
	}
}
