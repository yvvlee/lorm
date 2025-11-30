package lorm

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestInsertSingle(t *testing.T) {
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")
	if driver == "" || dsn == "" {
		driver = "mysql"
		dsn = "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	}
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	m := &Test{Int: 777, Str: "single_insert", Timestamp: testTime, Datetime: testTime, Decimal: decimal.NewFromFloat(3.21), IntSlice: []int{7}, Struct: Sub{ID: 7, Name: "s"}}
	rows, err := Insert(ctx, e, m)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m.ID > 0)
}
