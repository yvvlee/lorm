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

func TestRetainedColumnResultsRejectRawBytes(t *testing.T) {
	type RawAlias = sql.RawBytes
	type RawPointerAlias = *sql.RawBytes
	t.Run("raw", testRetainedColumnTypeRejected[sql.RawBytes])
	t.Run("alias", testRetainedColumnTypeRejected[RawAlias])
	t.Run("pointer", testRetainedColumnTypeRejected[*sql.RawBytes])
	t.Run("pointer_alias", testRetainedColumnTypeRejected[RawPointerAlias])
}

func testRetainedColumnTypeRejected[T any](t *testing.T) {
	r := newScriptedQueryRecorder()
	e := newScriptedEngine(t, r)
	ctx := context.Background()
	stmt := e.Query[*orderedScanCoverageModel]().Select("name")
	_, found, err := stmt.GetCol[T](ctx)
	require.ErrorContains(t, err, "use []byte instead")
	require.False(t, found)
	require.Empty(t, stmt.builder.GetColumns(), "rejected terminal call must reset the statement")
	_, err = stmt.Select("name").FindCols[T](ctx)
	require.ErrorContains(t, err, "use []byte instead")
	_, count, err := stmt.Select("name").PageCols[T](ctx, 1, 10)
	require.ErrorContains(t, err, "use []byte instead")
	require.Zero(t, count)
	require.Nil(t, r.LastQuery(), "reject the type before executing SQL, including a page count")
	var values []T
	require.ErrorContains(t, ScanCols[T](nil, &values), "use []byte instead")
}

func TestOwnedColumnTypeDistinguishesByteTypes(t *testing.T) {
	type ByteAlias = []byte
	type Bytes []byte
	type OwnedRaw sql.RawBytes
	require.NoError(t, validateOwnedColumnType[[]byte]())
	require.NoError(t, validateOwnedColumnType[ByteAlias]())
	require.NoError(t, validateOwnedColumnType[Bytes]())
	require.NoError(t, validateOwnedColumnType[OwnedRaw]())
	require.NoError(t, validateOwnedColumnType[*[]byte]())
}
