package lorm

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewEngineWithOptionsInitBranches(t *testing.T) {
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(
		driver,
		dsn,
		WithMaxIdleConns(1),
		WithMaxOpenConns(1),
		WithConnMaxLifetime(1*time.Second),
		WithConnMaxIdleTime(1*time.Second),
	)
	if err != nil {
		t.Skip("db not available")
		return
	}
	assert.NotNil(t, e)
	_ = e.Close()
}
