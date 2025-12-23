package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQueryModelPageBranches(t *testing.T) {
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

	_, _, err = Query[*Test](e).Page(ctx, 1, 0)
	assert.Error(t, err)

	list, total, err := Query[*Test](e).Where("id < ?", 0).Page(ctx, 1, 10)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, total)
	assert.Nil(t, list)

	_, total, err = Query[*Test](e).Where("id > ?", 0).Page(ctx, 100, 10)
	assert.Nil(t, err)
	assert.True(t, total > 0)

	list, total, err = Query[*Test](e).Where("id > ?", 0).Page(ctx, 1, 1)
	assert.Nil(t, err)
	assert.True(t, total > 0)
	assert.True(t, len(list) <= 1)
}
