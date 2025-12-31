package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestInsertSingle(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	m := &Test{
		Int:       777,
		Str:       "single_insert",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(3.21),
		IntSlice:  []int{7},
		Struct:    Sub{ID: 7, Name: "s"},
	}
	rows, err := Insert[*Test](engine).AddModel(m).Exec(ctx)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m.ID > 0)
}

func TestInsertAllEmpty(t *testing.T) {
	var models []*Test
	rows, err := Insert[*Test](&Engine{config: &Config{}}).AddModels(models...).Exec(context.TODO())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, rows)
}

func TestInsertIgnore(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)

	m1 := &Test{
		Int:       888,
		Str:       "insert_ignore_test",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(8.88),
		IntSlice:  []int{8},
		Struct:    Sub{ID: 8, Name: "s8"},
	}

	rows, err := Insert[*Test](engine).AddModel(m1).Exec(ctx)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m1.ID > 0)

	originalID := m1.ID

	m2 := &Test{
		ID:        originalID,
		Int:       999,
		Str:       "should_be_ignored",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(9.99),
		IntSlice:  []int{9},
		Struct:    Sub{ID: 9, Name: "s9"},
	}

	rows, err = Insert[*Test](engine).Ignore().AddModel(m2).Exec(ctx)
	assert.Nil(t, err)

	result, err := Query[*Test](engine).Where("id = ?", originalID).Get(ctx)
	assert.Nil(t, err)
	assert.Equal(t, int(888), result.Int)
	assert.Equal(t, "insert_ignore_test", result.Str)
}

func TestInsertIgnoreAll(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)

	m1 := &Test{
		Int:       111,
		Str:       "batch_insert_1",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(1.11),
		IntSlice:  []int{1},
		Struct:    Sub{ID: 1, Name: "s1"},
	}

	rows, err := Insert[*Test](engine).AddModel(m1).Exec(ctx)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)

	originalID := m1.ID

	models := []*Test{
		{
			ID:        originalID,
			Int:       222,
			Str:       "should_be_ignored",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(2.22),
			IntSlice:  []int{2},
			Struct:    Sub{ID: 2, Name: "s2"},
		},
		{
			Int:       333,
			Str:       "batch_insert_2",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(3.33),
			IntSlice:  []int{3},
			Struct:    Sub{ID: 3, Name: "s3"},
		},
	}

	rows, err = Insert[*Test](engine).Ignore().AddModels(models...).Exec(ctx)
	assert.Nil(t, err)

	result, err := Query[*Test](engine).Where("id = ?", originalID).Get(ctx)
	assert.Nil(t, err)
	assert.Equal(t, int(111), result.Int)
	assert.Equal(t, "batch_insert_1", result.Str)
}
