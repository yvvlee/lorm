package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionProxyBranches(t *testing.T) {
	e := &Engine{config: &Config{}}
	s := e.session(context.TODO())

	// proxy when no tx
	p := s.proxy()
	assert.Nil(t, p)
}

func TestSessionExecQueryExistBranches(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	engine := newScriptedEngine(t, recorder)
	sess := &session{engine: engine}

	_, err := sess.Exec(context.Background(), "DELETE FROM t")
	require.NoError(t, err)

	recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(1)})
	rows, err := sess.Query(context.Background(), "SELECT id FROM t WHERE id = ?", 1)
	require.NoError(t, err)
	_ = rows.Close()

	recorder.QueueQueryRows([]string{"value"}, []driver.Value{int64(1)})
	exists, err := sess.Exist(context.Background(), "SELECT 1")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSessionPlaceholderErrors(t *testing.T) {
	engine := newScriptedEngine(t, newScriptedQueryRecorder())
	engine.config.placeholderFormat = errorPlaceholder{}
	sess := &session{engine: engine}

	_, err := sess.Exec(context.Background(), "SELECT ?", 1)
	assert.ErrorContains(t, err, "replace failed")

	_, err = sess.Query(context.Background(), "SELECT ?", 1)
	assert.ErrorContains(t, err, "replace failed")

	_, err = sess.Exist(context.Background(), "SELECT ?", 1)
	assert.ErrorContains(t, err, "replace failed")
}
