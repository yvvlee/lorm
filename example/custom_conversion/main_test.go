package main

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestCustomConversionFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if exampleutil.IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	report := &Report{
		Title:  "quarterly-report",
		Scores: CSVInts{90, 95, 88},
	}
	_, err = engine.Insert[*Report]().AddModel(report).Exec(ctx)
	assert.NoError(t, err)

	repo := engine.Repository[*Report]()
	loaded, err := repo.Get(ctx, report.ID)
	assert.NoError(t, err)
	assert.Equal(t, CSVInts{90, 95, 88}, loaded.Scores)

	var r Report
	_, err = engine.Update[*Report]().
		ID(report.ID).
		SetMap(map[string]any{r.Fields().Scores(): CSVInts{100, 99, 98}}).
		Exec(ctx)
	assert.NoError(t, err)

	reloaded, err := repo.Get(ctx, report.ID)
	assert.NoError(t, err)
	assert.Equal(t, CSVInts{100, 99, 98}, reloaded.Scores)
}

func TestCSVIntsConversionBranches(t *testing.T) {
	encoded, err := CSVInts{1, 2, 3}.Value()
	assert.NoError(t, err)
	assert.Equal(t, driver.Value([]byte("1,2,3")), encoded)

	var empty CSVInts
	err = empty.Scan([]byte{})
	assert.NoError(t, err)
	assert.Nil(t, empty)

	var fromString CSVInts
	err = fromString.Scan("4,5")
	assert.NoError(t, err)
	assert.Equal(t, CSVInts{4, 5}, fromString)

	err = (*CSVInts)(nil).Scan([]byte("1,2"))
	assert.ErrorContains(t, err, "destination is nil")
}
