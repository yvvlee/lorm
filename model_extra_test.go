package lorm

import (
	"database/sql/driver"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

func TestModelToInsertData(t *testing.T) {
	m := &Test{}
	cols, vals := ModelToInsertData(m)
	assert.NotEmpty(t, cols)
	assert.Len(t, cols, len(m.Fields().All()))
	assert.NotEmpty(t, vals)
}

func TestModelsToInsertDataTransformsFields(t *testing.T) {
	m := &Test{
		Int:      7,
		Str:      "insert-data",
		IntSlice: []int{1, 2, 3},
		Struct:   Sub{ID: 1, Name: "json"},
	}

	cols, vals := ModelToInsertData(m, "id")
	assert.NotContains(t, cols, "id")
	assert.False(t, m.CreatedAt.IsZero())
	assert.False(t, m.UpdatedAt.IsZero())

	indexByColumn := make(map[string]int, len(cols))
	for i, col := range cols {
		indexByColumn[col] = i
	}

	assert.Same(t, &m.Str, vals[indexByColumn["str"]])
	assert.Same(t, &m.CreatedAt, vals[indexByColumn["created_at"]])
	assert.Same(t, &m.UpdatedAt, vals[indexByColumn["updated_at"]])

	intSliceWrapper, ok := vals[indexByColumn["int_slice"]].(*JSONFieldWrapper)
	assert.True(t, ok)
	assert.Same(t, &m.IntSlice, intSliceWrapper.v)

	structWrapper, ok := vals[indexByColumn["struct"]].(*JSONFieldWrapper)
	assert.True(t, ok)
	assert.Same(t, &m.Struct, structWrapper.v)
}

func TestGeneratedModelLormFieldPtr(t *testing.T) {
	m := &Test{}

	assert.Same(t, &m.Str, m.LormFieldPtr("str"))
	assert.Nil(t, m.LormFieldPtr("missing"))

	intSliceWrapper, ok := m.LormFieldPtr("int_slice").(*JSONFieldWrapper)
	assert.True(t, ok)
	assert.Same(t, &m.IntSlice, intSliceWrapper.v)

	structWrapper, ok := m.LormFieldPtr("struct").(*JSONFieldWrapper)
	assert.True(t, ok)
	assert.Same(t, &m.Struct, structWrapper.v)
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

	w2 := NewJSONFieldWrapper(nil)
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

	// unsupported type
	err = w.Scan(123)
	assert.Error(t, err)
}

func TestJSONFieldWrapperDelegatesDatabaseInterfaces(t *testing.T) {
	wrapped := NewJSONFieldWrapper(jsonWrapperValuer{value: "stored"})
	value, err := wrapped.Value()
	assert.NoError(t, err)
	assert.Equal(t, driver.Value("stored"), value)

	scanner := &jsonWrapperScanner{}
	err = NewJSONFieldWrapper(scanner).Scan([]byte(`{"a":1}`))
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"a":1}`), scanner.value)
}

func TestUnimplementedMarkers(t *testing.T) {
	UnimplementedModel{}.mustEmbedUnimplementedModel()
	UnimplementedTable{}.mustEmbedUnimplementedModel()
	UnimplementedTable{}.mustEmbedUnimplementedTable()
}

func TestJSONFieldWrapperValue(t *testing.T) {
	// nil value
	w := NewJSONFieldWrapper(nil)
	v, err := w.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	// non-nil value
	data := map[string]int{"a": 1}
	w = NewJSONFieldWrapper(data)
	v, err = w.Value()
	assert.NoError(t, err)
	assert.NotNil(t, v)
}

func TestAdaptDBArgsNilNestedDriverValuerPointer(t *testing.T) {
	var decimalValue *decimal.Decimal
	adapted := adaptDBArgs([]any{&decimalValue})
	assert.Len(t, adapted, 1)
	assert.Nil(t, adapted[0])
}

func TestAdaptDBArgsKeepsNonNilDriverValuerPointer(t *testing.T) {
	decimalValue := decimal.NewFromFloat(4.20)
	decimalPtr := &decimalValue
	adapted := adaptDBArgs([]any{&decimalPtr})
	assert.Len(t, adapted, 1)
	assert.Same(t, decimalPtr, adapted[0])
}
