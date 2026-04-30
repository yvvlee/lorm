package lorm

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFillCurrentTimeAllBranches(t *testing.T) {
	now := time.Unix(1000, 0)
	{
		var v time.Time
		fillCurrentTime(&v, now)
		assert.False(t, v.IsZero())
	}
	{
		var v int64
		fillCurrentTime(&v, now)
		assert.Equal(t, now.Unix(), v)
	}
	{
		var v uint64
		fillCurrentTime(&v, now)
		assert.EqualValues(t, uint64(now.Unix()), v)
	}
	{
		var v int32
		fillCurrentTime(&v, now)
		assert.EqualValues(t, int32(now.Unix()), v)
	}
	{
		var v uint32
		fillCurrentTime(&v, now)
		assert.EqualValues(t, uint32(now.Unix()), v)
	}
	{
		var v int
		fillCurrentTime(&v, now)
		assert.EqualValues(t, int(now.Unix()), v)
	}
	{
		var v string
		fillCurrentTime(&v, now)
		assert.NotEmpty(t, v)
	}
}

func TestFillCurrentTimeDoublePointerBranches(t *testing.T) {
	now := time.Unix(1000, 0)

	t.Run("TimePointer", func(t *testing.T) {
		var v *time.Time
		fillCurrentTime(&v, now)
		assert.NotNil(t, v)
		assert.Equal(t, now, *v)
	})

	t.Run("Int64Pointer", func(t *testing.T) {
		var v *int64
		fillCurrentTime(&v, now)
		assert.NotNil(t, v)
		assert.Equal(t, now.Unix(), *v)
	})

	t.Run("StringPointer", func(t *testing.T) {
		var v *string
		fillCurrentTime(&v, now)
		assert.NotNil(t, v)
		assert.Equal(t, now.Format(time.DateTime), *v)
	})
}

func TestFillCurrentTimeSupportsNullTime(t *testing.T) {
	now := time.Unix(1000, 0)

	t.Run("Value", func(t *testing.T) {
		var v sql.NullTime
		fillCurrentTime(&v, now)
		assert.True(t, v.Valid)
		assert.Equal(t, now, v.Time)
	})

	t.Run("Pointer", func(t *testing.T) {
		var v *sql.NullTime
		fillCurrentTime(&v, now)
		if assert.NotNil(t, v) {
			assert.True(t, v.Valid)
			assert.Equal(t, now, v.Time)
		}
	})
}

func TestNewUpdatedFieldValueDoublePointerBranches(t *testing.T) {
	now := time.Unix(1000, 0)

	t.Run("TimePointer", func(t *testing.T) {
		var v *time.Time
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		assert.Equal(t, &now, updatedValue)
		syncValue()
		assert.NotNil(t, v)
		assert.Equal(t, now, *v)
	})

	t.Run("StringPointer", func(t *testing.T) {
		var v *string
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		expected := now.Format(time.DateTime)
		assert.Equal(t, &expected, updatedValue)
		syncValue()
		assert.NotNil(t, v)
		assert.Equal(t, expected, *v)
	})
}

func TestNewUpdatedFieldValueSupportsNullTime(t *testing.T) {
	now := time.Unix(1000, 0)

	t.Run("Value", func(t *testing.T) {
		var v sql.NullTime
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		expected := sql.NullTime{Time: now, Valid: true}
		assert.Equal(t, &expected, updatedValue)
		syncValue()
		assert.Equal(t, expected, v)
	})

	t.Run("Pointer", func(t *testing.T) {
		var v *sql.NullTime
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		expected := sql.NullTime{Time: now, Valid: true}
		assert.Equal(t, &expected, updatedValue)
		syncValue()
		if assert.NotNil(t, v) {
			assert.Equal(t, expected, *v)
		}
	})
}

func TestIncrementVersionValueDoublePointerBranches(t *testing.T) {
	t.Run("AllocatesNilPointer", func(t *testing.T) {
		var v *int64
		incrementVersionValue(&v)
		assert.NotNil(t, v)
		assert.EqualValues(t, 1, *v)
	})

	t.Run("IncrementsExistingPointer", func(t *testing.T) {
		current := uint32(4)
		v := &current
		incrementVersionValue(&v)
		assert.EqualValues(t, 5, *v)
	})
}

type _pkInt64 struct {
	UnimplementedTable
	ID int64
}

func (m *_pkInt64) TableName() string { return "test" }
func (m *_pkInt64) New() Model        { return new(_pkInt64) }
func (m *_pkInt64) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkInt64) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkUint32 struct {
	UnimplementedTable
	ID uint32
}

func (m *_pkUint32) TableName() string { return "test" }
func (m *_pkUint32) New() Model        { return new(_pkUint32) }
func (m *_pkUint32) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkUint32) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkInt16 struct {
	UnimplementedTable
	ID int16
}

func (m *_pkInt16) TableName() string { return "test" }
func (m *_pkInt16) New() Model        { return new(_pkInt16) }
func (m *_pkInt16) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkInt16) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkUint8 struct {
	UnimplementedTable
	ID uint8
}

func (m *_pkUint8) TableName() string { return "test" }
func (m *_pkUint8) New() Model        { return new(_pkUint8) }
func (m *_pkUint8) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkUint8) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkUint struct {
	UnimplementedTable
	ID uint
}

func (m *_pkUint) TableName() string { return "test" }
func (m *_pkUint) New() Model        { return new(_pkUint) }
func (m *_pkUint) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkUint) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkInt struct {
	UnimplementedTable
	ID int
}

func (m *_pkInt) TableName() string { return "test" }
func (m *_pkInt) New() Model        { return new(_pkInt) }
func (m *_pkInt) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkInt) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkInt32 struct {
	UnimplementedTable
	ID int32
}

func (m *_pkInt32) TableName() string { return "test" }
func (m *_pkInt32) New() Model        { return new(_pkInt32) }
func (m *_pkInt32) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkInt32) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkUint16 struct {
	UnimplementedTable
	ID uint16
}

func (m *_pkUint16) TableName() string { return "test" }
func (m *_pkUint16) New() Model        { return new(_pkUint16) }
func (m *_pkUint16) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkUint16) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type _pkInt64Ptr struct {
	UnimplementedTable
	ID *int64
}

func (m *_pkInt64Ptr) TableName() string { return "test" }
func (m *_pkInt64Ptr) New() Model        { return new(_pkInt64Ptr) }
func (m *_pkInt64Ptr) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}
func (m *_pkInt64Ptr) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{Fields: []*FieldDescriptor{{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement}}}
}

type fakeResult struct{ id int64 }

func (f fakeResult) LastInsertId() (int64, error) { return f.id, nil }
func (f fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeResultErr struct{ err error }

func (f fakeResultErr) LastInsertId() (int64, error) { return 0, f.err }
func (f fakeResultErr) RowsAffected() (int64, error) { return 1, nil }

func TestFillModelIDAllTypeBranches(t *testing.T) {
	{
		m := &_pkInt64{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, 123, m.ID)
	}
	{
		m := &_pkUint32{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, uint32(123), m.ID)
	}
	{
		m := &_pkInt16{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, int16(123), m.ID)
	}
	{
		m := &_pkUint8{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, uint8(123), m.ID)
	}
	{
		m := &_pkUint{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, uint(123), m.ID)
	}
	{
		m := &_pkInt{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, int(123), m.ID)
	}
	{
		m := &_pkInt32{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, int32(123), m.ID)
	}
	{
		m := &_pkUint16{}
		_ = fillModelID(m, fakeResult{id: 123})
		assert.EqualValues(t, uint16(123), m.ID)
	}
	{
		m := &_pkInt64Ptr{}
		err := fillModelID(m, fakeResult{id: 123})
		assert.NoError(t, err)
		if assert.NotNil(t, m.ID) {
			assert.EqualValues(t, 123, *m.ID)
		}
	}
}

func TestFillModelIDSkipsExistingAndPropagatesErrors(t *testing.T) {
	existing := &_pkInt64{ID: 99}
	err := fillModelID(existing, fakeResult{id: 123})
	assert.NoError(t, err)
	assert.EqualValues(t, 99, existing.ID)

	err = fillModelID(&_pkInt64{}, fakeResultErr{err: assert.AnError})
	assert.ErrorIs(t, err, assert.AnError)

	manual := &manualPrimaryKeyModel{ID: "manual"}
	err = fillModelID(manual, fakeResult{id: 123})
	assert.NoError(t, err)
	assert.Equal(t, "manual", manual.ID)
}

func TestFillModelPrimaryKeyUnsupportedType(t *testing.T) {
	value := ""
	err := fillModelPrimaryKey(&value, 7)
	assert.ErrorContains(t, err, "unsupported primary key type")
}

func TestFillModelPrimaryKeyRejectsOverflow(t *testing.T) {
	t.Run("Uint8Overflow", func(t *testing.T) {
		var v uint8
		err := fillModelPrimaryKey(&v, 256)
		assert.ErrorContains(t, err, "overflows")
		assert.EqualValues(t, 0, v)
	})

	t.Run("Int8Overflow", func(t *testing.T) {
		var v int8
		err := fillModelPrimaryKey(&v, 128)
		assert.ErrorContains(t, err, "overflows")
		assert.EqualValues(t, 0, v)
	})

	t.Run("Int8Underflow", func(t *testing.T) {
		var v int8
		err := fillModelPrimaryKey(&v, -129)
		assert.ErrorContains(t, err, "overflows")
		assert.EqualValues(t, 0, v)
	})

	t.Run("Uint64Negative", func(t *testing.T) {
		var v uint64
		err := fillModelPrimaryKey(&v, -1)
		assert.ErrorContains(t, err, "overflows")
		assert.EqualValues(t, 0, v)
	})
}

func TestFillModelPrimaryKeySupportsDoublePointers(t *testing.T) {
	var intValue *int
	err := fillModelPrimaryKey(&intValue, 7)
	assert.NoError(t, err)
	if assert.NotNil(t, intValue) {
		assert.EqualValues(t, 7, *intValue)
	}

	var uintValue *uint64
	err = fillModelPrimaryKey(&uintValue, 8)
	assert.NoError(t, err)
	if assert.NotNil(t, uintValue) {
		assert.EqualValues(t, 8, *uintValue)
	}
}

func TestFillCurrentTimeRemainingBranches(t *testing.T) {
	now := time.Unix(2000, 0)

	t.Run("Uint64Pointer", func(t *testing.T) {
		var v *uint64
		fillCurrentTime(&v, now)
		if assert.NotNil(t, v) {
			assert.EqualValues(t, uint64(now.Unix()), *v)
		}
	})

	t.Run("Int32Pointer", func(t *testing.T) {
		var v *int32
		fillCurrentTime(&v, now)
		if assert.NotNil(t, v) {
			assert.EqualValues(t, int32(now.Unix()), *v)
		}
	})

	t.Run("Uint32Pointer", func(t *testing.T) {
		var v *uint32
		fillCurrentTime(&v, now)
		if assert.NotNil(t, v) {
			assert.EqualValues(t, uint32(now.Unix()), *v)
		}
	})

	t.Run("IntPointer", func(t *testing.T) {
		var v *int
		fillCurrentTime(&v, now)
		if assert.NotNil(t, v) {
			assert.EqualValues(t, int(now.Unix()), *v)
		}
	})

	t.Run("ExistingPointerPreservesValue", func(t *testing.T) {
		current := "preset"
		ptr := &current
		fillCurrentTime(&ptr, now)
		assert.Equal(t, "preset", *ptr)
	})
}

func TestNewUpdatedFieldValueRemainingBranches(t *testing.T) {
	now := time.Unix(3000, 0)

	t.Run("Int64Pointer", func(t *testing.T) {
		var v *int64
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		assert.EqualValues(t, now.Unix(), *updatedValue.(*int64))
		syncValue()
		if assert.NotNil(t, v) {
			assert.EqualValues(t, now.Unix(), *v)
		}
	})

	t.Run("Uint64Pointer", func(t *testing.T) {
		var v *uint64
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		assert.EqualValues(t, uint64(now.Unix()), *updatedValue.(*uint64))
		syncValue()
		if assert.NotNil(t, v) {
			assert.EqualValues(t, uint64(now.Unix()), *v)
		}
	})

	t.Run("Uint32Pointer", func(t *testing.T) {
		var v *uint32
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		assert.EqualValues(t, uint32(now.Unix()), *updatedValue.(*uint32))
		syncValue()
		if assert.NotNil(t, v) {
			assert.EqualValues(t, uint32(now.Unix()), *v)
		}
	})

	t.Run("IntPointer", func(t *testing.T) {
		var v *int
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		assert.EqualValues(t, int(now.Unix()), *updatedValue.(*int))
		syncValue()
		if assert.NotNil(t, v) {
			assert.EqualValues(t, int(now.Unix()), *v)
		}
	})

	t.Run("StringValue", func(t *testing.T) {
		var v string
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.True(t, ok)
		expected := now.Format(time.DateTime)
		assert.Equal(t, expected, *updatedValue.(*string))
		syncValue()
		assert.Equal(t, expected, v)
	})

	t.Run("Unsupported", func(t *testing.T) {
		var v bool
		updatedValue, syncValue, ok := newUpdatedFieldValue(&v, now)
		assert.False(t, ok)
		assert.Nil(t, updatedValue)
		assert.Nil(t, syncValue)
	})
}

func TestIncrementVersionValueRemainingBranches(t *testing.T) {
	t.Run("IntValue", func(t *testing.T) {
		v := 1
		incrementVersionValue(&v)
		assert.EqualValues(t, 2, v)
	})

	t.Run("Uint16Value", func(t *testing.T) {
		v := uint16(9)
		incrementVersionValue(&v)
		assert.EqualValues(t, 10, v)
	})

	t.Run("Uint8Pointer", func(t *testing.T) {
		var v *uint8
		incrementVersionValue(&v)
		if assert.NotNil(t, v) {
			assert.EqualValues(t, 1, *v)
		}
	})
}

func TestToolsAdditionalBranchCoverage(t *testing.T) {
	now := time.Unix(1000, 0)

	var updatedInt32 int32
	updatedValue, syncValue, ok := newUpdatedFieldValue(&updatedInt32, now)
	assert.True(t, ok)
	assert.EqualValues(t, int32(now.Unix()), *updatedValue.(*int32))
	syncValue()
	assert.EqualValues(t, int32(now.Unix()), updatedInt32)

	currentUint8 := uint8(4)
	incrementVersionValue(&currentUint8)
	assert.EqualValues(t, 5, currentUint8)

	var currentUint16 *uint16
	incrementVersionValue(&currentUint16)
	if assert.NotNil(t, currentUint16) {
		assert.EqualValues(t, 1, *currentUint16)
	}

	var updatedInt64 int64
	updatedValue, syncValue, ok = newUpdatedFieldValue(&updatedInt64, now)
	assert.True(t, ok)
	assert.EqualValues(t, now.Unix(), *updatedValue.(*int64))
	syncValue()
	assert.EqualValues(t, now.Unix(), updatedInt64)

	var updatedUint64 uint64
	updatedValue, syncValue, ok = newUpdatedFieldValue(&updatedUint64, now)
	assert.True(t, ok)
	assert.EqualValues(t, uint64(now.Unix()), *updatedValue.(*uint64))
	syncValue()
	assert.EqualValues(t, uint64(now.Unix()), updatedUint64)

	var updatedUint32 uint32
	updatedValue, syncValue, ok = newUpdatedFieldValue(&updatedUint32, now)
	assert.True(t, ok)
	assert.EqualValues(t, uint32(now.Unix()), *updatedValue.(*uint32))
	syncValue()
	assert.EqualValues(t, uint32(now.Unix()), updatedUint32)

	var updatedInt int
	updatedValue, syncValue, ok = newUpdatedFieldValue(&updatedInt, now)
	assert.True(t, ok)
	assert.EqualValues(t, int(now.Unix()), *updatedValue.(*int))
	syncValue()
	assert.EqualValues(t, int(now.Unix()), updatedInt)

	currentInt8 := int8(1)
	incrementVersionValue(&currentInt8)
	assert.EqualValues(t, 2, currentInt8)

	currentInt16 := int16(2)
	incrementVersionValue(&currentInt16)
	assert.EqualValues(t, 3, currentInt16)

	currentInt32 := int32(3)
	incrementVersionValue(&currentInt32)
	assert.EqualValues(t, 4, currentInt32)

	currentInt64 := int64(4)
	incrementVersionValue(&currentInt64)
	assert.EqualValues(t, 5, currentInt64)

	currentUint := uint(5)
	incrementVersionValue(&currentUint)
	assert.EqualValues(t, 6, currentUint)

	currentUint8Value := uint8(6)
	incrementVersionValue(&currentUint8Value)
	assert.EqualValues(t, 7, currentUint8Value)

	currentUint32 := uint32(7)
	incrementVersionValue(&currentUint32)
	assert.EqualValues(t, 8, currentUint32)

	currentUint64 := uint64(8)
	incrementVersionValue(&currentUint64)
	assert.EqualValues(t, 9, currentUint64)

	var currentInt *int
	incrementVersionValue(&currentInt)
	if assert.NotNil(t, currentInt) {
		assert.EqualValues(t, 1, *currentInt)
	}

	var currentInt8Ptr *int8
	incrementVersionValue(&currentInt8Ptr)
	if assert.NotNil(t, currentInt8Ptr) {
		assert.EqualValues(t, 1, *currentInt8Ptr)
	}

	var currentInt16Ptr *int16
	incrementVersionValue(&currentInt16Ptr)
	if assert.NotNil(t, currentInt16Ptr) {
		assert.EqualValues(t, 1, *currentInt16Ptr)
	}

	var currentInt32Ptr *int32
	incrementVersionValue(&currentInt32Ptr)
	if assert.NotNil(t, currentInt32Ptr) {
		assert.EqualValues(t, 1, *currentInt32Ptr)
	}

	var currentUintPtr *uint
	incrementVersionValue(&currentUintPtr)
	if assert.NotNil(t, currentUintPtr) {
		assert.EqualValues(t, 1, *currentUintPtr)
	}
}
