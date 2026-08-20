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
	rows, err := (&Engine{config: &Config{}}).Insert[*Test]().AddModels(models...).Exec(context.TODO())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, rows)
}

func TestInsertTypedNilReturnsErrorFromExec(t *testing.T) {
	stmt := (&Engine{config: &Config{}}).Insert[*Test]().AddModel(nil)

	_, err := stmt.Exec(context.Background())
	assert.ErrorContains(t, err, "model at index 0 is nil")
}

func TestInsertStmtInternalBuilderWrappers(t *testing.T) {
	stmt := (&Engine{
		config: &Config{
			Dialect: DialectConfig{
				PlaceholderFormat: builder.Question,
				Escaper:           names.NoEscaper,
			},
		},
	}).Insert[*Test]().
		Prefix("WITH audit AS ?", 0).
		PrefixExpr(builder.Expr("/* insert */")).
		columns("id", "str").
		values(1, "wrapped").
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
		engine.config.Dialect.SupportsReturning = true
		engine.config.Dialect.SupportsLastInsertID = false

		model := &conversionModel{Name: "alpha", Codes: csvInts{1, 2}}
		rowsAffected, err := engine.Insert[*conversionModel]().AddModel(model).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
		assert.EqualValues(t, 41, model.ID)

		call := recorder.LastQuery()
		require.NotNil(t, call)
		assert.Contains(t, call.query, "RETURNING id")
	})

	t.Run("batchDoesNotUseReturningByDefault", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.Dialect.SupportsReturning = true
		engine.config.Dialect.SupportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := engine.Insert[*conversionModel]().AddModels(models...).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
		assert.Zero(t, models[0].ID)
		assert.Zero(t, models[1].ID)
		assert.Nil(t, recorder.LastQuery())
		call := recorder.LastExec()
		require.NotNil(t, call)
		assert.NotContains(t, call.query, "RETURNING")
	})

	t.Run("requiredBatchBackfillExecutesOneByOne", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.Dialect.SupportsReturning = true
		engine.config.Dialect.SupportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := engine.Insert[*conversionModel]().RequireIDBackfill().AddModels(models...).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 2, rowsAffected)
		assert.EqualValues(t, 1, models[0].ID)
		assert.EqualValues(t, 1, models[1].ID)
		assert.Equal(t, 2, recorder.QueryCallCount())
	})
}

func TestInsertRequireIDBackfillUsesTransactionAndSharedTime(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	models := []*Test{{Str: "first"}, {Str: "second"}}

	rowsAffected, err := engine.Insert[*Test]().
		RequireIDBackfill().
		AddModels(models...).
		Exec(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 2, rowsAffected)
	assert.Len(t, recorder.BeginTxCalls(), 1)
	assert.Len(t, recorder.Calls(), 2)
	assert.NotZero(t, models[0].ID)
	assert.NotZero(t, models[1].ID)
	assert.False(t, models[0].CreatedAt.IsZero())
	assert.Equal(t, models[0].CreatedAt, models[1].CreatedAt)
	assert.Equal(t, models[0].UpdatedAt, models[1].UpdatedAt)
}

func TestInsertRequireIDBackfillResetsAfterExec(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	stmt := engine.Insert[*reservedWordModel]().RequireIDBackfill()

	_, err := stmt.AddModels(
		&reservedWordModel{Group: "first"},
		&reservedWordModel{Group: "second"},
	).Exec(context.Background())
	require.NoError(t, err)
	assert.Len(t, recorder.Calls(), 2)

	recorder.Reset()
	_, err = stmt.AddModels(
		&reservedWordModel{Group: "third"},
		&reservedWordModel{Group: "fourth"},
	).Exec(context.Background())
	require.NoError(t, err)
	assert.Len(t, recorder.Calls(), 1)
	assert.Empty(t, recorder.BeginTxCalls())
}

func TestInsertRequireIDBackfillFailsWithoutDriverSupport(t *testing.T) {
	engine := &Engine{config: &Config{driverName: "unsupported"}}
	_, err := engine.Insert[*reservedWordModel]().
		RequireIDBackfill().
		AddModels(&reservedWordModel{}, &reservedWordModel{}).
		Exec(context.Background())
	assert.ErrorContains(t, err, "required ID backfill is not supported")
}

func TestInsertAutoPrimaryKeyColumnPolicy(t *testing.T) {
	t.Run("singleExplicitIDIsInsertedAndPreserved", func(t *testing.T) {
		recorder := newCaptureSQLRecorder()
		engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
		model := &reservedWordModel{ID: 42, Group: "explicit"}

		rows, err := engine.Insert[*reservedWordModel]().AddModel(model).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rows)
		assert.EqualValues(t, 42, model.ID)
		call := recorder.Last()
		assert.Equal(t, "INSERT INTO `order` (`id`,`group`) VALUES (?,?)", call.query)
		assert.Equal(t, []any{int64(42), "explicit"}, call.args)
	})

	t.Run("mixedBatchPreservesInputGroups", func(t *testing.T) {
		recorder := newCaptureSQLRecorder()
		engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
		models := []*reservedWordModel{
			{Group: "generated-first"},
			{ID: 42, Group: "explicit-first"},
			{ID: 43, Group: "explicit-second"},
			{Group: "generated-last"},
		}

		_, err := engine.Insert[*reservedWordModel]().AddModels(models...).Exec(context.Background())
		require.NoError(t, err)
		assert.Len(t, recorder.BeginTxCalls(), 1)
		calls := recorder.Calls()
		require.Len(t, calls, 3)
		assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?)", calls[0].query)
		assert.Equal(t, "INSERT INTO `order` (`id`,`group`) VALUES (?,?),(?,?)", calls[1].query)
		assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?)", calls[2].query)
		assert.Zero(t, models[0].ID)
		assert.EqualValues(t, 42, models[1].ID)
		assert.EqualValues(t, 43, models[2].ID)
		assert.Zero(t, models[3].ID)
	})

	t.Run("requiredBackfillHandlesMixedIDsOneByOne", func(t *testing.T) {
		recorder := newCaptureSQLRecorder()
		engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
		models := []*reservedWordModel{
			{Group: "generated-first"},
			{ID: 42, Group: "explicit"},
			{Group: "generated-last"},
		}

		rows, err := engine.Insert[*reservedWordModel]().
			RequireIDBackfill().
			AddModels(models...).
			Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 3, rows)
		assert.Len(t, recorder.Calls(), 3)
		assert.NotZero(t, models[0].ID)
		assert.EqualValues(t, 42, models[1].ID)
		assert.NotZero(t, models[2].ID)
	})
}
