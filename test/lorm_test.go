package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm"

	//nolint:revive
	_ "github.com/go-sql-driver/mysql"
)

func TestEngine(t *testing.T) {
	engine, err := lorm.NewEngine("mysql", "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local")
	assert.Nil(t, err)
	defer engine.Close()
	ctx := context.TODO()
	list, err := lorm.Query[*Test](engine).
		From("test").
		Where("id < ?", 3).
		Find(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(list))
	single, err := lorm.Query[*Test](engine).
		From("test").
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	assert.Nil(t, err)
	assert.Equal(t, single.ID, uint64(1))
	ids, err := lorm.QueryCol[uint64](engine).
		From("test").
		Columns("id").
		Where("id < ?", 3).
		Find(ctx)
	assert.Nil(t, err)
	assert.Equal(t, ids, []uint64{1, 2})
	id, exist, err := lorm.QueryCol[uint64](engine).
		From("test").
		Columns("id").
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	assert.Nil(t, err)
	assert.True(t, exist)
	assert.Equal(t, id, uint64(1))
}
