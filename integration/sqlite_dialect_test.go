package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/yvvlee/lorm"
)

// TestSQLiteRuntimeReturningBackfill verifies both strategies against SQLite.
// Parameters: t manages the test lifecycle.
// Returns: none.
// Throws: none.
func TestSQLiteRuntimeReturningBackfill(t *testing.T) {
	t.Run("detected RETURNING", func(t *testing.T) {
		engine := newSQLiteSemanticsEngine(t, WithLogger(nil))
		defer engine.Close()
		require.True(t, engine.SupportsReturning())
		assertSQLiteGeneratedID(t, engine)
	})
	t.Run("explicit LastInsertId", func(t *testing.T) {
		engine := newSQLiteSemanticsEngine(t, WithLogger(nil), WithSupportsReturning(false))
		defer engine.Close()
		require.False(t, engine.SupportsReturning())
		assertSQLiteGeneratedID(t, engine)
	})
}

func assertSQLiteGeneratedID(t *testing.T, engine *Engine) {
	t.Helper()
	ctx := context.Background()
	model := &updateSemanticsModel{
		Name:    "detected dialect",
		Version: 1,
	}
	rows, err := engine.Insert[*updateSemanticsModel]().AddModel(model).Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)
	require.Positive(t, model.ID)

	loaded, err := engine.Repository[*updateSemanticsModel]().Get(ctx, model.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, model.Name, loaded.Name)
}
