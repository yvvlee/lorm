package lorm

import (
	"database/sql"
	"time"

	"github.com/spf13/cast"
)

func fillModelID(table Table, result sql.Result) error {
	descriptor := table.LormModelDescriptor()
	primaryKeys := descriptor.FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	if len(primaryKeys) != 1 {
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
