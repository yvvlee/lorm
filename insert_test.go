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
	m := &Test{Int: 777, Str: "single_insert", Timestamp: testTime, Datetime: testTime, Decimal: decimal.NewFromFloat(3.21), IntSlice: []int{7}, Struct: Sub{ID: 7, Name: "s"}}
	rows, err := Insert(ctx, engine, m)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m.ID > 0)
}

func TestInsertAllEmpty(t *testing.T) {
	var models []*Test
	rows, err := InsertAll(context.TODO(), &Engine{config: &Config{}}, models)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, rows)
}
