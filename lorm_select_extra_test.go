package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestQueryModelExistBranches(t *testing.T) {
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

	ex, err := Query[*Test](e).Where("id > ?", 0).Exist(ctx)
	assert.Nil(t, err)
	assert.True(t, ex)

	ex, err = Query[*Test](e).Where("id < ?", 0).Exist(ctx)
	assert.Nil(t, err)
	assert.False(t, ex)
}

func TestQueryColGetFalseAndError(t *testing.T) {
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

	_, ok, err := QueryCol[uint64](e).From("test").Columns("id").Where("id < ?", 0).Limit(1).Get(ctx)
	assert.Nil(t, err)
	assert.False(t, ok)

	_, _, err = QueryCol[uint64](e).Prefix("INVALID").From("test").Columns("id").Limit(1).Get(ctx)
	assert.Error(t, err)
}
