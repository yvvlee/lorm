package lorm

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonWrapperValuer struct {
	value driver.Value
	err   error
}

func (v jsonWrapperValuer) Value() (driver.Value, error) {
	return v.value, v.err
}

type jsonWrapperScanner struct {
	value any
	err   error
}

func (s *jsonWrapperScanner) Scan(src any) error {
	s.value = src
	return s.err
}

func TestGeneratedInsertHookTransformsFields(t *testing.T) {
	m := &Test{
		Int:      7,
		Str:      "insert-data",
		IntSlice: []int{1, 2, 3},
		Struct:   Sub{ID: 1, Name: "json"},
	}

	plan := m.LormBeforeInsert(time.Now())
	cols, vals := plan.Columns, plan.Values
	assert.NotContains(t, cols, "id")
	assert.False(t, m.CreatedAt.IsZero())
	assert.False(t, m.UpdatedAt.IsZero())

	indexByColumn := make(map[string]int, len(cols))
	for i, col := range cols {
		indexByColumn[col] = i
	}

	assert.Equal(t, m.Str, vals[indexByColumn["str"]])
	assert.Equal(t, m.CreatedAt, vals[indexByColumn["created_at"]])
	assert.Equal(t, m.UpdatedAt, vals[indexByColumn["updated_at"]])
	assert.Nil(t, vals[indexByColumn["decimal_p"]])

	intSliceWrapper, ok := vals[indexByColumn["int_slice"]].(*JSONFieldWrapper[[]int])
	assert.True(t, ok)
	assert.Same(t, &m.IntSlice, intSliceWrapper.target)

	structWrapper, ok := vals[indexByColumn["struct"]].(*JSONFieldWrapper[Sub])
	assert.True(t, ok)
	assert.Same(t, &m.Struct, structWrapper.target)
}

func TestGeneratedModelLormFieldPtr(t *testing.T) {
	m := &Test{}

	assert.Same(t, &m.Str, m.LormFieldPtr("str"))
	assert.Nil(t, m.LormFieldPtr("missing"))

	intSliceWrapper, ok := m.LormFieldPtr("int_slice").(*JSONFieldWrapper[[]int])
	assert.True(t, ok)
	assert.Same(t, &m.IntSlice, intSliceWrapper.target)

	structWrapper, ok := m.LormFieldPtr("struct").(*JSONFieldWrapper[Sub])
	assert.True(t, ok)
	assert.Same(t, &m.Struct, structWrapper.target)
}

func TestGeneratedInsertPlansShareReadOnlyColumns(t *testing.T) {
	firstZero := new(Test).LormBeforeInsert(time.Time{})
	secondZero := new(Test).LormBeforeInsert(time.Time{})
	require.NotEmpty(t, firstZero.Columns)
	require.Same(t, &firstZero.Columns[0], &secondZero.Columns[0])

	firstSet := (&Test{ID: 1}).LormBeforeInsert(time.Time{})
	secondSet := (&Test{ID: 2}).LormBeforeInsert(time.Time{})
	require.NotEmpty(t, firstSet.Columns)
	require.Same(t, &firstSet.Columns[0], &secondSet.Columns[0])
	require.NotSame(t, &firstZero.Columns[0], &firstSet.Columns[0])
	require.Len(t, firstZero.Values, len(firstZero.Columns))
	require.Len(t, firstSet.Values, len(firstSet.Columns))
}

func TestHooklessInsertPlansShareColumns(t *testing.T) {
	first := &hooklessWriteModel{ID: "1", Name: "first"}
	columns := first.LormModelDescriptor().AllFields()
	firstPlan, err := prepareInsertPlan(first, time.Time{}, columns)
	require.NoError(t, err)
	secondPlan, err := prepareInsertPlan(&hooklessWriteModel{ID: "2", Name: "second"}, time.Time{}, columns)
	require.NoError(t, err)
	require.Same(t, &firstPlan.Columns[0], &secondPlan.Columns[0])
	require.Equal(t, []any{"1", "first"}, firstPlan.Values)
	require.Equal(t, []any{"2", "second"}, secondPlan.Values)
}

func TestGeneratedFieldsWithAliasDoesNotMutateDefault(t *testing.T) {
	m := &Test{}

	a := m.Fields().WithAlias("a")
	b := m.Fields().WithAlias("b")

	assert.Equal(t, "a.id", a.ID())
	assert.Equal(t, "b.id", b.ID())
	assert.Equal(t, "id", m.Fields().ID())
}

func TestJSONFieldWrapperStringAndUnmarshal(t *testing.T) {
	var v []int
	w := NewJSONFieldWrapper(&v)
	data, err := w.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, data)

	err = w.UnmarshalJSON([]byte(`[1,2,3]`))
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, v)

	s := w.String()
	assert.Contains(t, s, "1")

	w2 := NewJSONFieldWrapper[[]int](nil)
	assert.Equal(t, "", w2.String())
}

func TestJSONFieldWrapperScan(t *testing.T) {
	var obj struct {
		A int `json:"a"`
	}
	w := NewJSONFieldWrapper(&obj)
	err := w.Scan([]byte(`{"a":1}`))
	assert.NoError(t, err)
	assert.Equal(t, 1, obj.A)

	err = w.Scan("{\"a\":2}")
	assert.NoError(t, err)
	assert.Equal(t, 2, obj.A)

	err = w.Scan(nil)
	assert.NoError(t, err)
	assert.Zero(t, obj)

	// unsupported type
	err = w.Scan(123)
	assert.Error(t, err)
}

func TestJSONFieldWrapperDelegatesDatabaseInterfaces(t *testing.T) {
	valuer := jsonWrapperValuer{value: "stored"}
	wrapped := NewJSONFieldWrapper(&valuer)
	value, err := wrapped.Value()
	assert.NoError(t, err)
	assert.Equal(t, driver.Value("stored"), value)

	scanner := &jsonWrapperScanner{}
	err = NewJSONFieldWrapper(scanner).Scan([]byte(`{"a":1}`))
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"a":1}`), scanner.value)
}

func TestJSONFieldWrapperScanNullishDelegatesToDatabaseScanner(t *testing.T) {
	tests := []struct {
		name string
		src  any
	}{
		{name: "sql null", src: nil},
		{name: "string null", src: "null"},
		{name: "bytes null", src: []byte("null")},
		{name: "empty string", src: ""},
		{name: "empty bytes", src: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &jsonWrapperScanner{value: "old", err: assert.AnError}
			err := NewJSONFieldWrapper(scanner).Scan(tt.src)
			assert.ErrorIs(t, err, assert.AnError)
			assert.Equal(t, tt.src, scanner.value)
		})
	}
}

func TestUnimplementedMarkers(t *testing.T) {
	UnimplementedModel{}.mustEmbedUnimplementedModel()
	UnimplementedTable{}.mustEmbedUnimplementedModel()
	UnimplementedTable{}.mustEmbedUnimplementedTable()
}

func TestJSONFieldWrapperValue(t *testing.T) {
	// nil value
	nilWrapper := NewJSONFieldWrapper[map[string]int](nil)
	v, err := nilWrapper.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Error(t, nilWrapper.Scan("{}"))
	assert.Error(t, nilWrapper.Scan([]byte(`{}`)))
	assert.Error(t, nilWrapper.UnmarshalJSON([]byte(`{}`)))

	// non-nil value
	data := map[string]int{"a": 1}
	w := NewJSONFieldWrapper(&data)
	v, err = w.Value()
	assert.NoError(t, err)
	assert.NotNil(t, v)
}

func TestJSONFieldWrapperScanNullishWithNilTarget(t *testing.T) {
	nilWrapper := NewJSONFieldWrapper[map[string]int](nil)
	tests := []struct {
		name string
		src  any
	}{
		{name: "sql null", src: nil},
		{name: "string null", src: "null"},
		{name: "bytes null", src: []byte("null")},
		{name: "empty string", src: ""},
		{name: "empty bytes", src: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, nilWrapper.Scan(tt.src))
		})
	}
}

func TestJSONFieldWrapperScanNullishClearsTarget(t *testing.T) {
	tests := []struct {
		name string
		src  any
	}{
		{name: "sql null", src: nil},
		{name: "string null", src: "null"},
		{name: "bytes null", src: []byte("null")},
		{name: "empty string", src: ""},
		{name: "empty bytes", src: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			number := 42
			assert.NoError(t, NewJSONFieldWrapper(&number).Scan(tt.src))
			assert.Zero(t, number)

			slice := []int{1, 2}
			assert.NoError(t, NewJSONFieldWrapper(&slice).Scan(tt.src))
			assert.Nil(t, slice)

			mapping := map[string]int{"a": 1}
			assert.NoError(t, NewJSONFieldWrapper(&mapping).Scan(tt.src))
			assert.Nil(t, mapping)

			structure := Sub{ID: 1, Name: "old"}
			assert.NoError(t, NewJSONFieldWrapper(&structure).Scan(tt.src))
			assert.Zero(t, structure)

			pointer := &Sub{ID: 1}
			assert.NoError(t, NewJSONFieldWrapper(&pointer).Scan(tt.src))
			assert.Nil(t, pointer)
		})
	}
}

func TestJSONFieldWrapperScanRejectsInvalidNullSpellings(t *testing.T) {
	for _, spelling := range []string{"NULL", "Null", "Nil", "nil"} {
		t.Run(spelling, func(t *testing.T) {
			t.Run("string", func(t *testing.T) {
				assertJSONFieldWrapperScanErrorDoesNotClear(t, spelling)
			})
			t.Run("bytes", func(t *testing.T) {
				assertJSONFieldWrapperScanErrorDoesNotClear(t, []byte(spelling))
			})
		})
	}
}

func assertJSONFieldWrapperScanErrorDoesNotClear(t *testing.T, src any) {
	t.Helper()
	value := Sub{ID: 1, Name: "old"}
	assert.Error(t, NewJSONFieldWrapper(&value).Scan(src))
	assert.Equal(t, Sub{ID: 1, Name: "old"}, value)
}

func TestJSONFieldWrapperScanQuotedNullIsJSONString(t *testing.T) {
	value := "old"
	assert.NoError(t, NewJSONFieldWrapper(&value).Scan(`"null"`))
	assert.Equal(t, "null", value)
}
