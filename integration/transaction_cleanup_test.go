package integration

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm"
)

func TestTXGoexitRollsBackAndReleasesConnection(t *testing.T) {
	for _, logging := range []bool{false, true} {
		for _, nested := range []bool{false, true} {
			name := "no_logging"
			if logging {
				name = "logging"
			}
			if nested {
				name += "/nested"
			}
			t.Run(name, func(t *testing.T) {
				driver, dsn := mustIntegrationDriverAndDSN(t)
				var logger lorm.Logger
				if logging {
					logger = slog.New(slog.NewTextHandler(io.Discard, nil))
				}
				engine, err := lorm.NewEngine(driver, dsn, lorm.WithLogger(logger), lorm.WithMaxOpenConns(1))
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, engine.Close()) })
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				_, err = engine.Exec(ctx, "DROP TABLE IF EXISTS tx_cleanup_models")
				require.NoError(t, err)
				_, err = engine.Exec(ctx, "CREATE TABLE tx_cleanup_models (id BIGINT PRIMARY KEY)")
				require.NoError(t, err)
				var insertErr error
				var returned bool
				insertAndExit := func(txCtx context.Context) error {
					_, insertErr = engine.Exec(txCtx, "INSERT INTO tx_cleanup_models (id) VALUES (1)")
					if insertErr != nil {
						return insertErr
					}
					runtime.Goexit()
					return nil
				}
				done := make(chan struct{})
				go func() {
					defer close(done)
					_ = engine.TX(ctx, func(txCtx context.Context) error {
						if nested {
							return engine.TXWithOptions(txCtx, &sql.TxOptions{}, insertAndExit)
						}
						return insertAndExit(txCtx)
					})
					returned = true
				}()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("transaction goroutine did not exit")
				}
				require.NoError(t, insertErr)
				require.False(t, returned, "Goexit must still terminate the callback goroutine")
				// The original context remains active. A leaked connection would block this query.
				queryCtx, cancelQuery := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancelQuery()
				rows, err := engine.SQL(queryCtx, "SELECT COUNT(*) FROM tx_cleanup_models")
				require.NoError(t, err)
				defer rows.Close()
				var count int64
				require.NoError(t, lorm.ScanCol(rows, &count))
				assert.Zero(t, count, "the INSERT must have been rolled back")
			})
		}
	}
}
