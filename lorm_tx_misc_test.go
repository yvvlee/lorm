package lorm

import (
	"context"
	"errors"
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

func TestTXErrorBranch(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()
	// error branch: fn returns error
	err := e.TX(ctx, func(ctx context.Context) error { return assert.AnError })
	assert.Error(t, err)
}

func TestTXPanicRollsBack(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()

	assert.PanicsWithValue(t, "boom", func() {
		_ = engine.TX(context.Background(), func(context.Context) error {
			panic("boom")
		})
	})
}

func TestTXFailureBranchesCoverage(t *testing.T) {
	t.Run("beginError", func(t *testing.T) {
		engine := newTxBehaviorEngine(t, txBehavior{beginErr: errors.New("begin failed")})
		err := engine.TX(context.Background(), func(context.Context) error { return nil })
		assert.ErrorContains(t, err, "begin failed")
	})

	t.Run("rollbackError", func(t *testing.T) {
		engine := newTxBehaviorEngine(t, txBehavior{rollbackErr: errors.New("rollback failed")})
		err := engine.TX(context.Background(), func(context.Context) error { return assert.AnError })
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("commitError", func(t *testing.T) {
		engine := newTxBehaviorEngine(t, txBehavior{commitErr: errors.New("commit failed")})
		err := engine.TX(context.Background(), func(context.Context) error { return nil })
		assert.ErrorContains(t, err, "commit failed")
	})
}
