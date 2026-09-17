package lorm

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonDatabaseCodec struct {
	Text string `json:"text"`
}

func (*jsonDatabaseCodec) Scan(any) error               { return assert.AnError }
func (*jsonDatabaseCodec) Value() (driver.Value, error) { return nil, assert.AnError }

func TestJSONFieldWrapperUsesJSONInsteadOfDatabaseInterfaces(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		testJSONFieldDatabaseCodec[jsonDatabaseCodec](t, `{"text":""}`)
	})
	t.Run("pointer", func(t *testing.T) {
		testJSONFieldDatabaseCodec[*jsonDatabaseCodec](t, `null`)
	})
	t.Run("named_pointer", func(t *testing.T) {
		type pointer *jsonDatabaseCodec
		testJSONFieldDatabaseCodec[pointer](t, `null`)
	})
}

func testJSONFieldDatabaseCodec[T any](t *testing.T, nullJSON string) {
	var field T
	wrapper := NewJSONFieldWrapper(&field)
	for _, source := range []any{`{"text":"decoded"}`, []byte(`{"text":"decoded"}`)} {
		require.NoError(t, wrapper.Scan(source))
		value, err := wrapper.Value()
		require.NoError(t, err)
		assert.JSONEq(t, `{"text":"decoded"}`, string(value.([]byte)))
	}
	for _, source := range []any{nil, "null", []byte("null"), "", []byte{}} {
		require.NoError(t, wrapper.Scan(`{"text":"old"}`))
		require.NoError(t, wrapper.Scan(source))
		value, err := wrapper.Value()
		require.NoError(t, err)
		assert.JSONEq(t, nullJSON, string(value.([]byte)))
	}
	for _, source := range []any{"not JSON", int64(42)} {
		err := wrapper.Scan(source)
		require.Error(t, err)
		require.NotErrorIs(t, err, assert.AnError, "database scanner must not handle invalid JSON")
	}
}

type jsonCustomCodec struct{ jsonDatabaseCodec }

func (v *jsonCustomCodec) MarshalJSON() ([]byte, error) {
	if v.Text == "reject" {
		return nil, assert.AnError
	}
	return json.Marshal("json:" + v.Text)
}

func (v *jsonCustomCodec) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &v.Text); err != nil {
		return err
	}
	if v.Text == "reject" {
		return assert.AnError
	}
	return nil
}

func TestJSONFieldWrapperUsesCustomJSONInterfaces(t *testing.T) {
	t.Run("value", testJSONCustomCodec[jsonCustomCodec])
	t.Run("pointer", testJSONCustomCodec[*jsonCustomCodec])
}

func testJSONCustomCodec[T any](t *testing.T) {
	var field T
	wrapper := NewJSONFieldWrapper(&field)
	require.NoError(t, wrapper.Scan(`"decoded"`))
	value, err := wrapper.Value()
	require.NoError(t, err)
	assert.JSONEq(t, `"json:decoded"`, string(value.([]byte)))
	require.ErrorIs(t, wrapper.Scan(`"reject"`), assert.AnError)
	_, err = wrapper.Value()
	require.ErrorIs(t, err, assert.AnError)
}

type databasePointerCodec struct {
	text  string
	calls int
}

func (v *databasePointerCodec) Scan(src any) error {
	v.calls++
	if src == "reject" {
		return assert.AnError
	}
	v.text = src.(string)
	return nil
}

func (v *databasePointerCodec) Value() (driver.Value, error) {
	if v.text == "reject" {
		return nil, assert.AnError
	}
	return "sql:" + v.text, nil
}

func TestDatabaseSQLHandlesCustomPointerFields(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"value"}, []driver.Value{"native"}, []driver.Value{nil}, []driver.Value{"reject"})
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)
	rows, err := db.QueryContext(context.Background(), "SELECT value")
	require.NoError(t, err)
	defer rows.Close()
	var field *databasePointerCodec
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&field))
	require.NotNil(t, field)
	assert.Equal(t, "native", field.text)
	assert.Equal(t, 1, field.calls)
	value, err := driver.DefaultParameterConverter.ConvertValue(&field)
	require.NoError(t, err)
	assert.Equal(t, "sql:native", value)

	previous := field
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&field))
	require.Nil(t, field)
	assert.Equal(t, 1, previous.calls, "SQL NULL must not call the underlying scanner")
	require.True(t, rows.Next())
	require.ErrorIs(t, rows.Scan(&field), assert.AnError)
	field = &databasePointerCodec{text: "reject"}
	_, err = driver.DefaultParameterConverter.ConvertValue(&field)
	require.ErrorIs(t, err, assert.AnError)
}
