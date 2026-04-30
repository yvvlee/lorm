package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestInsertAllEmpty(t *testing.T) {
	var models []*Test
	rows, err := Insert[*Test](&Engine{config: &Config{}}).AddModels(models...).Exec(context.TODO())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, rows)
}

func TestInsertStmtBuilderWrappers(t *testing.T) {
	stmt := Insert[*Test](&Engine{
		config: &Config{
			placeholderFormat: builder.Question,
			escaper:           names.NoEscaper,
		},
	}).
		Prefix("WITH audit AS ?", 0).
		PrefixExpr(builder.Expr("/* insert */")).
		Columns("id", "str").
		Values(1, "wrapped").
		Suffix("ON CONFLICT DO NOTHING").
		SuffixExpr(builder.Expr("RETURNING id"))

	sql, args, err := stmt.builder.ToSql()
	assert.NoError(t, err)
	assert.Contains(t, sql, "WITH audit AS ?")
	assert.Contains(t, sql, "INSERT INTO test (id,str) VALUES (?,?)")
	assert.Contains(t, sql, "ON CONFLICT DO NOTHING")
	assert.Contains(t, sql, "RETURNING id")
	assert.Equal(t, []any{0, 1, "wrapped"}, args)
}

func TestInsertExecWithReturningCoverage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(41)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		model := &conversionModel{Name: "alpha", Codes: csvInts{1, 2}}
		rowsAffected, err := Insert[*conversionModel](engine).AddModel(model).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
		assert.EqualValues(t, 41, model.ID)

		call := recorder.LastQuery()
		require.NotNil(t, call)
		assert.Contains(t, call.query, "RETURNING id")
	})

	t.Run("mismatch", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := Insert[*conversionModel](engine).AddModels(models...).Exec(context.Background())
		assert.ErrorContains(t, err, "expected 2 returned rows, got 1")
		assert.Zero(t, rowsAffected)
	})

	t.Run("ignoreAllowsPartialRows", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := Insert[*conversionModel](engine).Ignore().AddModels(models...).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
	})
}
