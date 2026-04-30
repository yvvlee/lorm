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
