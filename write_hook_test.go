package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hooklessWriteModel struct {
	UnimplementedTable
	ID   string
	Name string
}

func (*hooklessWriteModel) TableName() string { return "hookless_write" }
func (*hooklessWriteModel) New() Model        { return new(hooklessWriteModel) }

func (m *hooklessWriteModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	default:
		return nil
	}
}

func (m *hooklessWriteModel) LormFieldValue(name string) any {
	switch name {
	case "id":
		return m.ID
	case "name":
		return m.Name
	default:
		return nil
	}
}

func (m *hooklessWriteModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "hooklessWriteModel",
		TableName: m.TableName(),
		Fields: []*FieldDescriptor{
			{DBField: "id"},
			{DBField: "name"},
		},
	}
}

func TestConvertGeneratedID(t *testing.T) {
	signed, err := ConvertGeneratedSignedID[int8](127, 8, "Model.ID")
	require.NoError(t, err)
	assert.Equal(t, int8(127), signed)

	_, err = ConvertGeneratedSignedID[int8](128, 8, "Model.ID")
	assert.ErrorContains(t, err, "overflows Model.ID")

	unsigned, err := ConvertGeneratedUnsignedID[uint16](65_535, 16, "Model.ID")
	require.NoError(t, err)
	assert.Equal(t, uint16(65_535), unsigned)

	_, err = ConvertGeneratedUnsignedID[uint64](-1, 64, "Model.ID")
	assert.ErrorContains(t, err, "overflows Model.ID")
}

func TestManagedTimeHelpers(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	nullTime := ManagedNullTime(now)
	assert.True(t, nullTime.Valid)
	assert.Equal(t, now, nullTime.Time)
	assert.Equal(t, now.Format(time.DateTime), ManagedTimeString(now))
}

func TestWriteSupportsTableWithoutHooks(t *testing.T) {
	model := &hooklessWriteModel{ID: "manual-1", Name: "alice"}
	_, hasBeforeInsert := any(model).(BeforeInsertHook)
	_, hasAfterInsert := any(model).(AfterInsertHook)
	_, hasBeforeUpdate := any(model).(BeforeUpdateHook)
	_, hasAfterUpdate := any(model).(AfterUpdateHook)
	assert.False(t, hasBeforeInsert)
	assert.False(t, hasAfterInsert)
	assert.False(t, hasBeforeUpdate)
	assert.False(t, hasAfterUpdate)

	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	rowsAffected, err := engine.Insert[*hooklessWriteModel]().
		AddModel(model).
		Exec(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	assert.Equal(t, "INSERT INTO `hookless_write` (`id`,`name`) VALUES (?,?)", recorder.Last().query)
	assert.Equal(t, []any{"manual-1", "alice"}, recorder.Last().args)

	recorder.Reset()
	model.Name = "bob"
	rowsAffected, err = engine.Update[*hooklessWriteModel]().
		Set("name", model.Name).
		Where("id = ?", model.ID).
		Exec(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	assert.Equal(t, "UPDATE `hookless_write` SET `name` = ? WHERE id = ?", recorder.Last().query)
	assert.Equal(t, []any{"bob", "manual-1"}, recorder.Last().args)
}
