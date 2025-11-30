package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestQueryModelFindErrorBranch(t *testing.T) {
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
	_, err = Query[*Test](e).Prefix("INVALID").Find(ctx)
	assert.Error(t, err)
}
