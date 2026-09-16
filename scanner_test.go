package lorm

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

type scanCoverageModel struct {
	UnimplementedModel
	ID   int64
	Name string
}

func (*scanCoverageModel) New() Model { return new(scanCoverageModel) }

func (m *scanCoverageModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	default:
		return nil
	}
}

func (*scanCoverageModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name: "scanCoverageModel",
		Fields: []*FieldDescriptor{
			{Name: "ID", FullName: "ID", DBField: "id"},
			{Name: "Name", FullName: "Name", DBField: "name"},
		},
	}
}

func TestScanColsPreallocatedSlice(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)

	recorder.QueueQueryRows(
		[]string{"name"},
		[]driver.Value{"alice"},
		[]driver.Value{"bob"},
	)

	rows, err := db.Query(`SELECT name FROM items ORDER BY name ASC`)
	require.NoError(t, err)
	defer rows.Close()

	values := []string{"", ""}
	err = ScanCols(rows, &values)
	require.ErrorContains(t, err, "empty destination slice")
}

func TestScanColsEmptySlice(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)

	recorder.QueueQueryRows(
		[]string{"name"},
		[]driver.Value{"alice"},
		[]driver.Value{"bob"},
	)

	rows, err := db.Query(`SELECT name FROM items ORDER BY name ASC`)
	require.NoError(t, err)
	defer rows.Close()

	var values []string
	err = ScanCols(rows, &values)
	require.NoError(t, err)
	require.Equal(t, []string{"alice", "bob"}, values)
}

func TestScannerHelpersCoverage(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)

	recorder.QueueQueryRows(
		[]string{"id", "ignored", "name", "ignored2"},
		[]driver.Value{int64(1), []byte("discard"), "alice", int64(9)},
		[]driver.Value{int64(2), nil, "bob", int64(10)},
	)
	rows, err := db.Query(`SELECT id, ignored, name, ignored2 FROM items`)
	require.NoError(t, err)
	defer rows.Close()

	var models []*scanCoverageModel
	err = ScanModels(rows, &models)
	require.NoError(t, err)
	require.Len(t, models, 2)
	require.EqualValues(t, 1, models[0].ID)
	require.Equal(t, "alice", models[0].Name)
	require.EqualValues(t, 2, models[1].ID)
	require.Equal(t, "bob", models[1].Name)

	recorder.QueueQueryRows([]string{"value"}, []driver.Value{int64(7)})
	row, err := db.Query(`SELECT 7`)
	require.NoError(t, err)
	defer row.Close()
	var value int
	err = ScanCol(row, &value)
	require.NoError(t, err)
	require.Equal(t, 7, value)

	recorder.QueueQueryRows([]string{"value"})
	emptyRow, err := db.Query(`SELECT 1 WHERE 1 = 0`)
	require.NoError(t, err)
	defer emptyRow.Close()
	err = ScanCol(emptyRow, &value)
	require.ErrorIs(t, err, sql.ErrNoRows)

	recorder.QueueQueryRows([]string{"left", "right"}, []driver.Value{int64(1), int64(2)})
	multiColRows, err := db.Query(`SELECT 1, 2`)
	require.NoError(t, err)
	defer multiColRows.Close()
	var values []int64
	err = ScanCols(multiColRows, &values)
	require.ErrorContains(t, err, "exactly one column")

	recorder.QueueQueryRows([]string{"left", "right"}, []driver.Value{int64(1), int64(2)})
	multiColRow, err := db.Query(`SELECT 1, 2`)
	require.NoError(t, err)
	defer multiColRow.Close()
	err = ScanCol(multiColRow, &value)
	require.ErrorContains(t, err, "exactly one column")
}

// Includes two unmapped columns to measure both destination slice reuse and
// discarded values without relying on a generated ordered scanner.
func BenchmarkScanModelsCustomColumns(b *testing.B) {
	recorder := newScriptedQueryRecorder()
	db, err := sql.Open(registerScriptedQueryDriver(recorder), "")
	require.NoError(b, err)
	b.Cleanup(func() { _ = db.Close() })
	result := scriptedQueryResult{columns: []string{"id", "ignored1", "name", "ignored2"}}
	for i := range 100 {
		result.rows = append(result.rows, []driver.Value{int64(i), "discard", "name", int64(i)})
	}
	b.ReportAllocs()
	for b.Loop() {
		recorder.results = append(recorder.results[:0], result)
		recorder.queryCalls = recorder.queryCalls[:0]
		rows, err := db.Query("SELECT id, ignored1, name, ignored2 FROM items")
		if err != nil {
			b.Fatal(err)
		}
		models, err := scanModelValues[*scanCoverageModel](rows)
		_ = rows.Close()
		if err != nil {
			b.Fatal(err)
		}
		if len(models) != 100 || models[99].ID != 99 || models[0].Name != "name" {
			b.Fatal("unexpected scanned models")
		}
	}
}
