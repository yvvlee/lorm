package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type jsonWrapperValuer struct {
	value driver.Value
	err   error
}

func (v jsonWrapperValuer) Value() (driver.Value, error) {
	return v.value, v.err
}

type jsonWrapperScanner struct {
	value any
	err   error
}

func (s *jsonWrapperScanner) Scan(src any) error {
	s.value = src
	return s.err
}

func TestJSONFieldWrapperDelegatesDatabaseInterfaces(t *testing.T) {
	wrapped := NewJSONFieldWrapper(jsonWrapperValuer{value: "stored"})
	value, err := wrapped.Value()
	assert.NoError(t, err)
	assert.Equal(t, driver.Value("stored"), value)

	scanner := &jsonWrapperScanner{}
	err = NewJSONFieldWrapper(scanner).Scan([]byte(`{"a":1}`))
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"a":1}`), scanner.value)
}

func TestEngineInitAppliesConfigAndFallbacks(t *testing.T) {
	skipUnlessSQLite3Available(t)

	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer db.Close()

	engine := &Engine{
		config: &Config{
			maxIdleConns:    2,
			maxOpenConns:    3,
			connMaxLifetime: time.Second,
			connMaxIdleTime: time.Second,
		},
		db: db,
	}
	engine.init()

	assert.Equal(t, 3, db.Stats().MaxOpenConnections)
	assert.Equal(t, builder.Question, (&Engine{config: &Config{}}).Placeholder())
	assert.Equal(t, "field", (&Engine{config: &Config{}}).Escaper().Escape("field"))
}

func TestTXPanicRollsBack(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()

	assert.PanicsWithValue(t, "boom", func() {
		_ = engine.TX(context.Background(), func(context.Context) error {
			panic("boom")
		})
	})
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

func TestRepositoryInsertIgnoreWrappers(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()

	ctx := context.Background()
	repo := NewRepository[*Test](engine)
	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)

	original := &Test{
		Int:       701,
		Str:       "repo_insert_ignore_original",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(7.01),
		IntSlice:  []int{7, 0, 1},
		Struct:    Sub{ID: 701, Name: "original"},
	}

	rows, err := repo.Insert(ctx, original)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, rows)

	ignored := &Test{
		ID:        original.ID,
		Int:       702,
		Str:       "repo_insert_ignore_single",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(7.02),
		IntSlice:  []int{7, 0, 2},
		Struct:    Sub{ID: 702, Name: "ignored"},
	}

	_, err = repo.InsertIgnore(ctx, ignored)
	assert.NoError(t, err)

	current, err := repo.Get(ctx, original.ID)
	assert.NoError(t, err)
	assert.Equal(t, "repo_insert_ignore_original", current.Str)

	batch := []*Test{
		{
			ID:        original.ID,
			Int:       703,
			Str:       "repo_insert_ignore_all_duplicate",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(7.03),
			IntSlice:  []int{7, 0, 3},
			Struct:    Sub{ID: 703, Name: "duplicate"},
		},
		{
			Int:       704,
			Str:       "repo_insert_ignore_all_inserted",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(7.04),
			IntSlice:  []int{7, 0, 4},
			Struct:    Sub{ID: 704, Name: "inserted"},
		},
	}

	_, err = repo.InsertIgnoreAll(ctx, batch)
	assert.NoError(t, err)

	inserted, err := repo.GetByField(ctx, "str", "repo_insert_ignore_all_inserted")
	assert.NoError(t, err)
	assert.NotNil(t, inserted)
	assert.NotZero(t, inserted.ID)
}
