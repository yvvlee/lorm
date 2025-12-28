package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQueryModelExistBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	// Prepare test data
	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	models := []*Test{
		{
			Int:       1,
			Bool:      true,
			Str:       "a",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(1.10),
			IntSlice:  []int{1, 2, 3},
			Struct:    Sub{ID: 1, Name: "haha"},
		},
		{
			Int:       2,
			Bool:      false,
			Str:       "b",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(2.12),
			IntSlice:  []int{1, 2, 3},
			Struct:    Sub{ID: 1, Name: "haha"},
		},
	}
	repo := NewTestRepository(e)
	_, err := repo.InsertAll(ctx, models)
	assert.Nil(t, err)

	ex, err := Query[*Test](e).Where("id > ?", 0).Exist(ctx)
	assert.Nil(t, err)
	assert.True(t, ex)

	ex, err = Query[*Test](e).Where("id < ?", 0).Exist(ctx)
	assert.Nil(t, err)
	assert.False(t, ex)
}

func TestQueryColGetFalseAndError(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, ok, err := QueryCol[uint64](e).
		Select("id").
		From("test").
		Where("id < ?", 0).
		Limit(1).
		Get(ctx)
	assert.Nil(t, err)
	assert.False(t, ok)

	_, _, err = QueryCol[uint64](e).Prefix("INVALID").
		Select("id").
		From("test").
		Limit(1).
		Get(ctx)
	assert.Error(t, err)
}
