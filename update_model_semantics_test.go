package lorm

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

func newSQLiteSemanticsEngine(t *testing.T) *Engine {
	t.Helper()
	skipUnlessSQLite3Available(t)

	engine, err := NewEngine(
		"sqlite3",
		filepath.Join(t.TempDir(), "semantics.sqlite"),
		WithMaxOpenConns(1),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = engine.Exec(ctx, `CREATE TABLE update_semantics_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version INTEGER NOT NULL,
		updated_at DATETIME NOT NULL
	)`)
	require.NoError(t, err)
	return engine
}

func TestUpdateSetModelRefreshesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	engine := newSQLiteSemanticsEngine(t)
	defer engine.Close()

	model := &updateSemanticsModel{Name: "draft", Version: 1}
	_, err := Insert[*updateSemanticsModel](engine).AddModel(model).Exec(ctx)
	require.NoError(t, err)

	repo := NewRepository[*updateSemanticsModel](engine)
	loaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	before := loaded.UpdatedAt

	time.Sleep(20 * time.Millisecond)

	loaded.Name = "published"
	rowsAffected, err := Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.True(t, loaded.UpdatedAt.After(before), "expected in-memory updated_at to advance")

	reloaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.True(t, reloaded.UpdatedAt.After(before), "expected persisted updated_at to advance")
}

func TestUpdateSetModelSyncsVersionBackToModel(t *testing.T) {
	ctx := context.Background()
	engine := newSQLiteSemanticsEngine(t)
	defer engine.Close()

	model := &updateSemanticsModel{Name: "draft", Version: 1}
	_, err := Insert[*updateSemanticsModel](engine).AddModel(model).Exec(ctx)
	require.NoError(t, err)

	repo := NewRepository[*updateSemanticsModel](engine)
	loaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), loaded.Version)

	loaded.Name = "published"
	rowsAffected, err := Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.Equal(t, int64(2), loaded.Version, "expected in-memory version to increment after successful update")

	loaded.Name = "published-again"
	rowsAffected, err = Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.Equal(t, int64(3), loaded.Version)

	reloaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), reloaded.Version)
}

func TestUpdateSetModelCoverage(t *testing.T) {
	engine := &Engine{config: &Config{escaper: names.NoEscaper}}
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
