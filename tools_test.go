package lorm

import (
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

type fakeResult struct{ id int64 }

func (f fakeResult) LastInsertId() (int64, error) { return f.id, nil }
func (f fakeResult) RowsAffected() (int64, error) { return 1, nil }

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
}
