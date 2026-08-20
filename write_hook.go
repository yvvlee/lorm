package lorm

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// HookTime is the timestamp shared by all hooks in one write operation.
type HookTime = time.Time

// BeforeInsertHook prepares model values and a stable insert plan.
type BeforeInsertHook interface {
	LormBeforeInsert(now HookTime) InsertPlan
}

// AfterInsertHook applies a generated primary key after a successful insert.
type AfterInsertHook interface {
	LormAfterInsert(result InsertResult) error
}

// BeforeUpdateHook prepares a full-model update without mutating the model.
type BeforeUpdateHook interface {
	LormBeforeUpdate(now HookTime) (UpdatePlan, error)
}

// AfterUpdateHook applies managed values after a successful update.
type AfterUpdateHook interface {
	LormAfterUpdate(now HookTime, rowsAffected int64)
}

// TablePointer constrains write models to pointers to their concrete struct type.
type TablePointer[M any] interface {
	Table
	*M
}

// ColumnValue is an ordered column/value pair produced by generated code.
type ColumnValue struct {
	Column string
	Value  any
}

// InsertPlan contains one model's insert columns and values in declaration order.
// Generated models share Columns across plans; callers must treat it as read-only.
type InsertPlan struct {
	Columns             []string
	Values              []any
	AutoIncrementColumn string
	AutoIncrementZero   bool
}

// InsertResult carries an unambiguous generated ID for one inserted model.
type InsertResult struct {
	RowsAffected   int64
	GeneratedID    int64
	HasGeneratedID bool
}

// UpdatePlan contains the ordered SET, WHERE, and increment operations for SetModel.
type UpdatePlan struct {
	Set             []ColumnValue
	Where           []ColumnValue
	Increment       []string
	PrimaryKeyCount int
}

func prepareInsertPlan(model Table, now HookTime, sharedColumns []string) (InsertPlan, error) {
	if hook, ok := any(model).(BeforeInsertHook); ok {
		return hook.LormBeforeInsert(now), nil
	}
	descriptor := model.LormModelDescriptor()
	if descriptor == nil {
		return InsertPlan{}, fmt.Errorf("lorm: model %T returned a nil descriptor", model)
	}
	accessor, ok := any(model).(ModelFieldValueAccessor)
	if !ok {
		return InsertPlan{}, fmt.Errorf("lorm: model %T does not implement ModelFieldValueAccessor", model)
	}

	if sharedColumns == nil {
		sharedColumns = descriptor.AllFields()
	}
	plan := InsertPlan{Columns: sharedColumns, Values: make([]any, 0, len(sharedColumns))}
	for _, field := range descriptor.Fields {
		if field == nil {
			return InsertPlan{}, fmt.Errorf("lorm: model %T has a nil field descriptor", model)
		}
		plan.Values = append(plan.Values, accessor.LormFieldValue(field.DBField))
	}
	return plan, nil
}

func prepareUpdatePlan(model Table, now HookTime) (UpdatePlan, error) {
	if hook, ok := any(model).(BeforeUpdateHook); ok {
		return hook.LormBeforeUpdate(now)
	}
	return UpdatePlan{}, nil
}

// ManagedNullTime builds a valid sql.NullTime for a managed time field.
func ManagedNullTime(now HookTime) sql.NullTime {
	return sql.NullTime{Time: now, Valid: true}
}

// ManagedTimeString formats a managed string time field.
func ManagedTimeString(now HookTime) string {
	return now.Format(time.DateTime)
}

// NilVersionError reports a nil pointer version field.
func NilVersionError(model, field string) error {
	return fmt.Errorf("lorm: %s.%s version field is nil", model, field)
}

type signedGeneratedID interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type unsignedGeneratedID interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// ConvertGeneratedSignedID converts a database-generated ID without reflection.
// A bits value of zero means the target-sized int width.
func ConvertGeneratedSignedID[T signedGeneratedID](value int64, bits int, field string) (T, error) {
	if bits == 0 {
		bits = strconv.IntSize
	}
	if bits < 64 {
		minValue := -int64(1 << (bits - 1))
		maxValue := int64(1<<(bits-1)) - 1
		if value < minValue || value > maxValue {
			var zero T
			return zero, fmt.Errorf("lorm: generated ID %d overflows %s", value, field)
		}
	}
	return T(value), nil
}

// ConvertGeneratedUnsignedID converts a database-generated ID without reflection.
// A bits value of zero means the target-sized uint width.
func ConvertGeneratedUnsignedID[T unsignedGeneratedID](value int64, bits int, field string) (T, error) {
	if value < 0 {
		var zero T
		return zero, fmt.Errorf("lorm: generated ID %d overflows %s", value, field)
	}
	if bits == 0 {
		bits = strconv.IntSize
	}
	if bits < 64 && uint64(value) > (uint64(1)<<bits)-1 {
		var zero T
		return zero, fmt.Errorf("lorm: generated ID %d overflows %s", value, field)
	}
	return T(value), nil
}
