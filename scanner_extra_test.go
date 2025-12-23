package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanRowErrorBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	// ModelScanner with unknown column triggers RawBytes error
	var m Test
	err := e.Query(ctx, NewModelScanner(&m), "SELECT 1 AS unknown")
	assert.Error(t, err)

	// ColsScanner with 2 columns triggers error
	var vv []int
	err = e.Query(ctx, NewColsScanner(&vv), "SELECT 1, 2")
	assert.Error(t, err)
}

func TestColScannerNoRows(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()
	var x int
	err := e.Query(ctx, NewColScanner(&x), "SELECT 1 WHERE 1=2")
	assert.Error(t, err)
}
