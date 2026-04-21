package lorm

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestScanColsPreallocatedSlice(t *testing.T) {
	skipUnlessSQLite3Available(t)
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE items (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO items(name) VALUES ('alice'), ('bob')`)
	require.NoError(t, err)

	rows, err := db.Query(`SELECT name FROM items ORDER BY name ASC`)
	require.NoError(t, err)
	defer rows.Close()

	values := []string{"", ""}
	err = ScanCols(rows, &values)
	require.NoError(t, err)
	require.Equal(t, []string{"alice", "bob"}, values)
}
