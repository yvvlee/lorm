package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOwnedByteColumnResultsSurviveBufferReuse(t *testing.T) {
	ctx := context.Background()
	recorder := newScriptedQueryRecorder()
	e := newScriptedEngine(t, recorder)
	queue := func() {
		recorder.QueueQueryRows([]string{"name"}, []driver.Value{[]byte("one")}, []driver.Value{[]byte("two")})
		recorder.results[len(recorder.results)-1].reuseBuffers = true
	}
	queue()
	value, found, err := e.Query[*orderedScanCoverageModel]().Select("name").GetCol[[]byte](ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("one"), value)
	queue()
	values, err := e.Query[*orderedScanCoverageModel]().Select("name").FindCols[[]byte](ctx)
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("one"), []byte("two")}, values)
	recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(2)})
	queue()
	page, count, err := e.Query[*orderedScanCoverageModel]().Select("name").PageCols[[]byte](ctx, 1, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Equal(t, [][]byte{[]byte("one"), []byte("two")}, page)

	type Bytes []byte
	queue()
	named, err := e.Query[*orderedScanCoverageModel]().Select("name").FindCols[Bytes](ctx)
	require.NoError(t, err)
	require.Equal(t, []Bytes{Bytes("one"), Bytes("two")}, named)

	// A single-row scan leaves ownership with the caller, so borrowed bytes are valid.
	queue()
	rows, err := e.SQL(ctx, "SELECT name")
	require.NoError(t, err)
	defer rows.Close()
	var raw sql.RawBytes
	require.NoError(t, ScanCol(rows, &raw))
	require.Equal(t, sql.RawBytes("one"), raw)
}
