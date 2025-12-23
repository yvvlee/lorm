package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryModelPageBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, _, err := Query[*Test](e).Page(ctx, 1, 0)
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
