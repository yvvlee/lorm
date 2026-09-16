package lorm

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTXCleansUpEveryExit(t *testing.T) {
	callbackErr := errors.New("callback failed")
	for _, logging := range []bool{false, true} {
		for _, nested := range []bool{false, true} {
			for _, exit := range []string{"commit", "error", "panic", "goexit"} {
				name := exit
				if logging {
					name += "/logging"
				} else {
					name += "/no_logging"
				}
				if nested {
					name += "/nested"
				}
				t.Run(name, func(t *testing.T) {
					var begins, commits, rollbacks atomic.Int64
					engine := newTxBehaviorEngine(t, txBehavior{beginCalls: &begins, commitCalls: &commits, rollbackCalls: &rollbacks})
					engine.db.SetMaxOpenConns(1)
					logger := new(recordingLogger)
					if logging {
						engine.logger = logger
					} else {
						engine.logger = nil
					}
					// Cancel only during cleanup, so context cancellation cannot hide a leaked transaction.
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					var returned bool
					var err error
					var recovered any
					fn := func(context.Context) error {
						switch exit {
						case "error":
							return callbackErr
						case "panic":
							panic(callbackErr)
						case "goexit":
							runtime.Goexit()
						}
						return nil
					}
					done := make(chan struct{})
					go func() {
						defer close(done)
						defer func() { recovered = recover() }()
						err = engine.TX(ctx, func(ctx context.Context) error {
							if nested {
								return engine.TX(ctx, fn)
							}
							return fn(ctx)
						})
						returned = true
					}()
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Fatal("transaction goroutine did not exit")
					}
					assert.EqualValues(t, 1, begins.Load())
					assert.Zero(t, engine.db.Stats().InUse)
					if exit == "commit" {
						assert.EqualValues(t, 1, commits.Load())
						assert.Zero(t, rollbacks.Load())
					} else {
						assert.Zero(t, commits.Load())
						assert.EqualValues(t, 1, rollbacks.Load())
					}
					assert.Equal(t, exit == "commit" || exit == "error", returned)
					if exit == "error" {
						assert.ErrorIs(t, err, callbackErr)
					} else {
						assert.NoError(t, err)
					}
					if exit == "panic" {
						assert.Same(t, callbackErr, recovered)
					} else {
						assert.Nil(t, recovered)
					}
					if logging {
						entries := logger.Entries()
						require.Len(t, entries, 2)
						want := "ROLLBACK"
						if exit == "commit" {
							want = "COMMIT"
						}
						if exit == "panic" {
							want = "ROLLBACK (panic)"
						}
						assert.Equal(t, want, entries[1].msg)
					}
				})
			}
		}
	}
}

func TestTXCleanupPreservesErrors(t *testing.T) {
	callbackErr, rollbackErr, commitErr := errors.New("callback failed"), errors.New("rollback failed"), errors.New("commit failed")
	for _, logging := range []bool{false, true} {
		for _, failCommit := range []bool{false, true} {
			behavior := txBehavior{rollbackErr: rollbackErr}
			if failCommit {
				behavior = txBehavior{commitErr: commitErr}
			}
			engine := newTxBehaviorEngine(t, behavior)
			if !logging {
				engine.logger = nil
			}
			err := engine.TX(context.Background(), func(context.Context) error {
				if failCommit {
					return nil
				}
				return callbackErr
			})
			if failCommit {
				assert.ErrorIs(t, err, commitErr)
			} else {
				assert.ErrorIs(t, err, callbackErr)
				assert.ErrorIs(t, err, rollbackErr)
			}
			assert.Zero(t, engine.db.Stats().InUse)
		}
	}
}
