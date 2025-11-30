package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScanRowErrorBranches(t *testing.T) {
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	ctx := context.TODO()

	// ModelScanner with unknown column triggers RawBytes error
	var m Test
	err = e.Query(ctx, NewModelScanner(&m), "SELECT 1 AS unknown")
	assert.Error(t, err)

	// ColsScanner with 2 columns triggers error
	var vv []int
	err = e.Query(ctx, NewColsScanner(&vv), "SELECT 1, 2")
	assert.Error(t, err)
}

func TestColScannerNoRows(t *testing.T) {
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	ctx := context.TODO()
	var x int
	err = e.Query(ctx, NewColScanner(&x), "SELECT 1 WHERE 1=2")
	assert.Error(t, err)
}
