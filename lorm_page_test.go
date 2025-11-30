package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestQueryModelPageBranches(t *testing.T) {
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
