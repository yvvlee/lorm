package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryModelExistBranches(t *testing.T) {
	e := initEngine(t)
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
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, ok, err := QueryCol[uint64](e).From("test").Columns("id").Where("id < ?", 0).Limit(1).Get(ctx)
	assert.Nil(t, err)
	assert.False(t, ok)

	_, _, err = QueryCol[uint64](e).Prefix("INVALID").From("test").Columns("id").Limit(1).Get(ctx)
	assert.Error(t, err)
}
