package lorm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/names"
)

type updateSemanticsModel struct {
	UnimplementedTable
	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Version   int64     `lorm:"version,version"`
	UpdatedAt time.Time `lorm:"updated_at,updated"`
}

func (m *updateSemanticsModel) TableName() string {
	return "update_semantics_models"
}

func (m *updateSemanticsModel) New() Model {
	return new(updateSemanticsModel)
}

func (m *updateSemanticsModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	case "version":
		return &m.Version
	case "updated_at":
		return &m.UpdatedAt
	default:
		return nil
	}
}

func (m *updateSemanticsModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "updateSemanticsModel",
		TableName: m.TableName(),
		Fields: []*FieldDescriptor{
			{Name: "ID", FullName: "ID", DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement},
			{Name: "Name", FullName: "Name", DBField: "name"},
			{Name: "Version", FullName: "Version", DBField: "version", Flag: FlagVersion},
			{Name: "UpdatedAt", FullName: "UpdatedAt", DBField: "updated_at", Flag: FlagUpdated},
		},
	}
}

func TestUpdateSetModelCoverage(t *testing.T) {
	engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: names.NoEscaper}}}
	model := &updateSemanticsModel{
		ID:        7,
		Name:      "published",
		Version:   3,
		UpdatedAt: time.Unix(100, 0),
	}

	stmt := Update[*updateSemanticsModel](engine).SetModel(model)
	require.NoError(t, stmt.err)

	sql, args, err := stmt.builder.ToSql()
	require.NoError(t, err)
	require.Len(t, args, 4)
	require.Equal(t, "UPDATE update_semantics_models SET name = ?, updated_at = ?, version = version+1 WHERE id = ? AND version = ?", sql)
	require.IsType(t, (*string)(nil), args[0])
	require.Equal(t, "published", *args[0].(*string))
	require.Equal(t, int64(7), args[2])
	require.Equal(t, int64(3), args[3])
}
