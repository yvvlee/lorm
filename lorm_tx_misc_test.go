package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTXExistingSessionBranch(t *testing.T) {
	e := &Engine{config: &Config{}}
	s := session{engine: e}
	ctx := context.WithValue(context.Background(), e, s)
	err := e.TX(ctx, func(ctx context.Context) error { return nil })
	assert.NoError(t, err)
}

func TestExecQueryExistErrorLogging(t *testing.T) {
	// Use actual engine to hit error path via invalid SQL
	// Reuse environment from lorm_test
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	ctx := context.TODO()

	_, err = e.Exec(ctx, "INVALID SQL")
	assert.Error(t, err)

	err = e.Query(ctx, NewColsScanner(&[]int{}), "INVALID SQL")
	assert.Error(t, err)

	_, err = e.Exist(ctx, "INVALID SQL")
	assert.Error(t, err)
}

func TestTXErrorBranchAndCommit(t *testing.T) {
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	ctx := context.TODO()
	// error branch: fn returns error
	err = e.TX(ctx, func(ctx context.Context) error { return assert.AnError })
	assert.Error(t, err)

	// explicit begin and commit branch
	s, err := e.beginTxSession(ctx)
	assert.NoError(t, err)
	assert.NoError(t, s.commit())

	// begin and close triggers rollback path
	s, err = e.beginTxSession(ctx)
	assert.NoError(t, err)
	assert.NoError(t, s.close())
}
