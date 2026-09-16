package lorm

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleResultQueriesReturnCloseErrors(t *testing.T) {
	closeErr := errors.New("query failed while closing rows")
	for _, tc := range []struct {
		name    string
		columns []string
		values  []driver.Value
		run     func(*testing.T, *Engine) error
	}{
		{"raw_exist", []string{"value"}, []driver.Value{int64(1)}, func(t *testing.T, e *Engine) error {
			found, err := e.Exist(context.Background(), "SELECT 1")
			require.False(t, found)
			return err
		}},
		{"exist", []string{"value"}, []driver.Value{int64(1)}, func(t *testing.T, e *Engine) error {
			found, err := e.Query[*orderedScanCoverageModel]().Exist(context.Background())
			require.False(t, found)
			return err
		}},
		{"get_ordered", []string{"id", "name"}, []driver.Value{int64(1), "name"}, func(t *testing.T, e *Engine) error {
			value, found, err := e.Query[*orderedScanCoverageModel]().Get(context.Background())
			require.False(t, found)
			require.Nil(t, value)
			return err
		}},
		{"get_named", []string{"id", "name"}, []driver.Value{int64(1), "name"}, func(t *testing.T, e *Engine) error {
			value, found, err := e.Query[*orderedScanCoverageModel]().Select("id", "name").Get(context.Background())
			require.False(t, found)
			require.Nil(t, value)
			return err
		}},
		{"get_col", []string{"id"}, []driver.Value{int64(1)}, func(t *testing.T, e *Engine) error {
			value, found, err := e.Query[*orderedScanCoverageModel]().Select("id").GetCol[int64](context.Background())
			require.False(t, found)
			require.Zero(t, value)
			return err
		}},
		{"count", []string{"count"}, []driver.Value{int64(1)}, func(t *testing.T, e *Engine) error {
			count, err := e.Query[*orderedScanCoverageModel]().Count(context.Background())
			require.Zero(t, count)
			return err
		}},
		{"page_count", []string{"count"}, []driver.Value{int64(1)}, func(t *testing.T, e *Engine) error {
			values, count, err := e.Query[*orderedScanCoverageModel]().Page(context.Background(), 1, 10)
			require.Nil(t, values)
			require.Zero(t, count)
			return err
		}},
	} {
		for _, logging := range []bool{false, true} {
			name := tc.name + "/without_logging"
			if logging {
				name = tc.name + "/with_logging"
			}
			t.Run(name, func(t *testing.T) {
				recorder := newScriptedQueryRecorder()
				recorder.QueueQueryRows(tc.columns, tc.values)
				recorder.results[0].closeErr = closeErr
				e := newScriptedEngine(t, recorder)
				if !logging {
					e.logger = nil
				}
				require.ErrorIs(t, tc.run(t, e), closeErr)
				require.Zero(t, e.db.Stats().InUse)
				require.Len(t, recorder.queryCalls, 1)
			})
		}
	}
}

func TestGetColPreservesScanAndCloseErrors(t *testing.T) {
	closeErr := errors.New("close failed")
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"id"}, []driver.Value{"not an integer"})
	recorder.results[0].closeErr = closeErr
	e := newScriptedEngine(t, recorder)
	_, found, err := e.Query[*orderedScanCoverageModel]().Select("id").GetCol[int64](context.Background())
	require.False(t, found)
	require.ErrorIs(t, err, closeErr)
	require.ErrorContains(t, err, "converting driver.Value")
}
