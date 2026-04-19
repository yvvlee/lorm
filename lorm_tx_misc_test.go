package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTXExistingSessionBranch(t *testing.T) {
	e := &Engine{config: &Config{}}
	s := &session{engine: e}
	ctx := context.WithValue(context.Background(), e, s)
	var called bool
	err := e.TX(ctx, func(ctx context.Context) error {
		called = true
		assert.Same(t, s, ctx.Value(e))
		return nil
	})
	assert.True(t, called)
	assert.NoError(t, err)
}

func TestExecQueryExistErrorLogging(t *testing.T) {
	// Use actual engine to hit error path via invalid SQL
	// Reuse environment from lorm_test
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, err := e.Exec(ctx, "INVALID SQL")
	assert.Error(t, err)

	_, err = e.Query(ctx, "INVALID SQL")
	assert.Error(t, err)

	_, err = e.Exist(ctx, "INVALID SQL")
	assert.Error(t, err)
}

func TestTXErrorBranchAndCommit(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()
	// error branch: fn returns error
	err := e.TX(ctx, func(ctx context.Context) error { return assert.AnError })
	assert.Error(t, err)

	// explicit begin and commit branch
	s, err := e.beginTxSession(ctx)
	assert.NoError(t, err)
	assert.NoError(t, s.commit())

	// begin and close triggers rollback path
	s, err = e.beginTxSession(ctx)
	assert.NoError(t, err)
	assert.NoError(t, s.close())
}
