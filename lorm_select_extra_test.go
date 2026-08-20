package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

type wrongNewModel struct {
	UnimplementedModel
	ID int64
}

type orderedScanCoverageModel struct {
	UnimplementedTable
	ID   int64
	Name string
}

var orderedScanCoverageCalls int
var orderedScanCoverageFieldPtrCalls int

func (*orderedScanCoverageModel) TableName() string { return "ordered_scan_coverage" }
func (*orderedScanCoverageModel) New() Model        { return new(orderedScanCoverageModel) }

func (m *orderedScanCoverageModel) LormScan(row RowScanner) error {
	orderedScanCoverageCalls++
	return row.Scan(&m.ID, &m.Name)
}

func (m *orderedScanCoverageModel) LormFieldPtr(name string) any {
	orderedScanCoverageFieldPtrCalls++
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	default:
		return nil
	}
}

func (*orderedScanCoverageModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "orderedScanCoverageModel",
		TableName: "ordered_scan_coverage",
		Fields: []*FieldDescriptor{
			{Name: "ID", FullName: "ID", DBField: "id"},
			{Name: "Name", FullName: "Name", DBField: "name"},
		},
	}
}

func (*wrongNewModel) New() Model { return new(reservedWordModel) }

func (m *wrongNewModel) LormFieldPtr(name string) any {
	if name == "id" {
		return &m.ID
	}
	return nil
}

func (*wrongNewModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:   "wrongNewModel",
		Fields: []*FieldDescriptor{{Name: "ID", FullName: "ID", DBField: "id"}},
	}
}

func TestSelectGetColSuccessCoverage(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(9)})
	engine := newScriptedEngine(t, recorder)

	value, ok, err := engine.Query[*conversionModel]().
		Select("id").
		Where("name = ?", "alpha").
		GetCol[int64](context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, 9, value)

	call := recorder.LastQuery()
	require.NotNil(t, call)
	assert.Equal(t, "SELECT id FROM conversion_models WHERE name = ? LIMIT 1", call.query)
	assert.Equal(t, []any{"alpha"}, call.args)
}

func TestSelectColumnTerminalEdgeCases(t *testing.T) {
	t.Run("getNull", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id"}, []driver.Value{nil})
		engine := newScriptedEngine(t, recorder)

		value, ok, err := engine.Query[*reservedWordModel]().
			Select("id").
			GetCol[*int64](context.Background())
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Nil(t, value)
	})

	t.Run("findEmpty", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id"})
		engine := newScriptedEngine(t, recorder)

		values, err := engine.Query[*reservedWordModel]().
			Select("id").
			FindCols[int64](context.Background())
		require.NoError(t, err)
		assert.Nil(t, values)
	})

	t.Run("getRejectsMultipleColumns", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(1), "staff"})
		engine := newScriptedEngine(t, recorder)

		_, _, err := engine.Query[*reservedWordModel]().
			Select("id", "group").
			GetCol[int64](context.Background())
		assert.ErrorContains(t, err, "exactly one column")
	})

	t.Run("getReturnsScanError", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id"}, []driver.Value{"not-an-integer"})
		engine := newScriptedEngine(t, recorder)

		_, _, err := engine.Query[*reservedWordModel]().
			Select("id").
			GetCol[int64](context.Background())
		assert.Error(t, err)
	})
}

func TestSelectRejectsWrongModelNewType(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(1)})
	engine := newScriptedEngine(t, recorder)

	_, _, err := engine.Query[*wrongNewModel]().
		Select("id").
		From("wrong_models").
		Get(context.Background())
	assert.ErrorContains(t, err, "Model.New returned")
}

func TestSelectUsesOrderedScannerOnlyForDefaultProjection(t *testing.T) {
	t.Run("defaultProjection", func(t *testing.T) {
		orderedScanCoverageCalls = 0
		orderedScanCoverageFieldPtrCalls = 0
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "name"}, []driver.Value{int64(1), "alice"})
		engine := newScriptedEngine(t, recorder)

		model, ok, err := engine.Query[*orderedScanCoverageModel]().Get(context.Background())
		require.NoError(t, err)
		require.True(t, ok)
		assert.EqualValues(t, 1, model.ID)
		assert.Equal(t, "alice", model.Name)
		assert.Equal(t, 1, orderedScanCoverageCalls)
		assert.Zero(t, orderedScanCoverageFieldPtrCalls)
	})

	t.Run("customProjection", func(t *testing.T) {
		orderedScanCoverageCalls = 0
		orderedScanCoverageFieldPtrCalls = 0
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"name", "id"}, []driver.Value{"bob", int64(2)})
		engine := newScriptedEngine(t, recorder)

		model, ok, err := engine.Query[*orderedScanCoverageModel]().
			Select("name", "id").
			Get(context.Background())
		require.NoError(t, err)
		require.True(t, ok)
		assert.EqualValues(t, 2, model.ID)
		assert.Equal(t, "bob", model.Name)
		assert.Zero(t, orderedScanCoverageCalls)
		assert.Equal(t, 2, orderedScanCoverageFieldPtrCalls)
	})
}

func TestSelectModelGetAndFindCoverage(t *testing.T) {
	t.Run("getSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(3), "ops"})
		engine := newScriptedEngine(t, recorder)

		model, ok, err := engine.Query[*reservedWordModel]().
			Where(builder.Eq{"group": "ops"}).
			Get(context.Background())
		require.NoError(t, err)
		require.True(t, ok)
		require.NotNil(t, model)
		assert.EqualValues(t, 3, model.ID)
		assert.Equal(t, "ops", model.Group)
		call := recorder.LastQuery()
		require.NotNil(t, call)
		assert.Equal(t, "SELECT id, group FROM order WHERE group = ? LIMIT 1", call.query)
		assert.Equal(t, []any{"ops"}, call.args)
	})

	t.Run("findSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows(
			[]string{"id", "group"},
			[]driver.Value{int64(1), "a"},
			[]driver.Value{int64(2), "b"},
		)
		engine := newScriptedEngine(t, recorder)

		list, err := engine.Query[*reservedWordModel]().
			Where("id > ?", 0).
			Find(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.EqualValues(t, 1, list[0].ID)
		assert.Equal(t, "a", list[0].Group)
		assert.EqualValues(t, 2, list[1].ID)
		assert.Equal(t, "b", list[1].Group)
	})

	t.Run("removeColumnsRestoresModelColumns", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(4), "ops"})
		engine := newScriptedEngine(t, recorder)

		model, ok, err := engine.Query[*reservedWordModel]().
			Select("id").
			RemoveColumns().
			Get(context.Background())
		require.NoError(t, err)
		require.True(t, ok)
		assert.EqualValues(t, 4, model.ID)
		assert.Equal(t, "ops", model.Group)

		call := recorder.LastQuery()
		require.NotNil(t, call)
		assert.Equal(t, "SELECT id, group FROM order LIMIT 1", call.query)
	})

	t.Run("queryColFindSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(1)}, []driver.Value{int64(2)})
		engine := newScriptedEngine(t, recorder)

		list, err := engine.Query[*reservedWordModel]().
			Select("id").
			From("reserved_word_models").
			FindCols[int64](context.Background())
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, list)
	})
}
