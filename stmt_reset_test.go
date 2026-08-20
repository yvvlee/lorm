package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestSelectStmtModelResetsAfterTerminalMethod(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Query[*reservedWordModel]().Where(builder.Eq{"group": "first"})
	_, err := stmt.Find(ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.Where(builder.Eq{"id": int64(2)}).Find(ctx)
	require.NoError(t, err)

	call := recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE `id` = ?", call.query)
	assert.Equal(t, []any{int64(2)}, call.args)
}

func TestSelectStmtColumnResetsAfterTerminalMethod(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Query[*manualPrimaryKeyModel]().
		Select("id").
		Where("id = ?", "manual-1")
	_, err := stmt.FindCols[string](ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.
		Select("name").
		Where("name = ?", "alice").
		FindCols[string](ctx)
	require.NoError(t, err)

	call := recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT name FROM `manual_keys` WHERE name = ?", call.query)
	assert.Equal(t, []any{"alice"}, call.args)
}

func TestInsertStmtResetsAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Insert[*reservedWordModel]().Ignore()
	rowsAffected, err := stmt.AddModel(&reservedWordModel{Group: "first"}).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	recorder.Reset()
	rowsAffected, err = stmt.AddModel(&reservedWordModel{Group: "second"}).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	call := recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?)", call.query)
	assert.Equal(t, []any{"second"}, call.args)
}

func TestDeleteStmtResetsAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Delete[*reservedWordModel]().
		Where(builder.Eq{"group": "first"}).
		Limit(1)
	rowsAffected, err := stmt.Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	recorder.Reset()
	rowsAffected, err = stmt.Where(builder.Eq{"id": int64(2)}).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	call := recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "DELETE FROM `order` WHERE `id` = ?", call.query)
	assert.Equal(t, []any{int64(2)}, call.args)
}

func TestDeleteStmtResetsAllowGlobalWriteAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	stmt := engine.Delete[*reservedWordModel]().AllowGlobalWrite()

	_, err := stmt.Exec(context.Background())
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause")
	assert.Empty(t, recorder.Calls())
}

func TestUpdateStmtResetsAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Update[*reservedWordModel]().
		SetMap(map[string]any{"group": "first"}).
		ID(1)
	rowsAffected, err := stmt.Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	recorder.Reset()
	rowsAffected, err = stmt.
		SetMap(map[string]any{"group": "second"}).
		Where(builder.Eq{"id": int64(2)}).
		Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	call := recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "UPDATE `order` SET `group` = ? WHERE `id` = ?", call.query)
	assert.Equal(t, []any{"second", int64(2)}, call.args)
}

func TestUpdateStmtResetsAllowGlobalWriteAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	stmt := engine.Update[*reservedWordModel]().Set("group", "first").AllowGlobalWrite()

	_, err := stmt.Exec(context.Background())
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.Set("group", "second").Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause")
	assert.Empty(t, recorder.Calls())
}

func TestUpdateStmtResetsAfterCallbacks(t *testing.T) {
	ctx := context.Background()
	engine := newConversionTestEngine(t, newConversionRecorder())

	stmt := engine.Update[*updateSemanticsModel]()
	model := &updateSemanticsModel{ID: 1, Name: "draft", Version: 1}
	model.Name = "published"
	rowsAffected, err := stmt.SetModel(model).Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)
	require.EqualValues(t, 2, model.Version)

	model.Name = "published-again"
	rowsAffected, err = stmt.SetModel(model).Exec(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)
	require.EqualValues(t, 3, model.Version)
}

func TestSelectStmtModelCloneExecDoesNotResetSource(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Query[*reservedWordModel]().Where(builder.Eq{"group": "staff"})
	clone := stmt.Clone().Where(builder.Eq{"id": int64(7)})
	_, err := clone.Find(ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.Find(ctx)
	require.NoError(t, err)

	call := recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE `group` = ?", call.query)
	assert.Equal(t, []any{"staff"}, call.args)
}

func TestSelectStmtColumnCloneExecDoesNotResetSource(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	stmt := engine.Query[*manualPrimaryKeyModel]().
		Select("id").
		Where("name = ?", "alice")
	clone := stmt.Clone().Where("id = ?", "manual-1")
	_, err := clone.FindCols[int64](ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = stmt.FindCols[int64](ctx)
	require.NoError(t, err)

	call := recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT id FROM `manual_keys` WHERE name = ?", call.query)
	assert.Equal(t, []any{"alice"}, call.args)
}

func TestWriteStmtCloneExecDoesNotResetSource(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	ctx := context.Background()

	updateStmt := engine.Update[*reservedWordModel]().
		SetMap(map[string]any{"group": "base"}).
		Where(builder.Eq{"id": int64(1)})
	updateClone := updateStmt.Clone().Where(builder.Eq{"group": "clone"})
	_, err := updateClone.Exec(ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = updateStmt.Exec(ctx)
	require.NoError(t, err)
	call := recorder.Last()
	assert.Equal(t, "UPDATE `order` SET `group` = ? WHERE `id` = ?", call.query)
	assert.Equal(t, []any{"base", int64(1)}, call.args)

	deleteStmt := engine.Delete[*reservedWordModel]().Where(builder.Eq{"id": int64(1)})
	deleteClone := deleteStmt.Clone().Where(builder.Eq{"group": "clone"})
	_, err = deleteClone.Exec(ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = deleteStmt.Exec(ctx)
	require.NoError(t, err)
	call = recorder.Last()
	assert.Equal(t, "DELETE FROM `order` WHERE `id` = ?", call.query)
	assert.Equal(t, []any{int64(1)}, call.args)

	insertStmt := engine.Insert[*reservedWordModel]().AddModel(&reservedWordModel{Group: "base"})
	insertClone := insertStmt.Clone().AddModel(&reservedWordModel{Group: "clone"})
	_, err = insertClone.Exec(ctx)
	require.NoError(t, err)

	recorder.Reset()
	_, err = insertStmt.Exec(ctx)
	require.NoError(t, err)
	call = recorder.Last()
	assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?)", call.query)
	assert.Equal(t, []any{"base"}, call.args)
}
