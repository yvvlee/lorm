package lorm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestModelToInsertData(t *testing.T) {
	m := &Test{}
	cols, vals := ModelToInsertData(m)
	assert.NotEmpty(t, cols)
	assert.Len(t, cols, len(m.Fields().All()))
	assert.NotEmpty(t, vals)
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
