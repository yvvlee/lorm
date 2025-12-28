package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
)

func TestDeleteWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = Delete(e).
		From("test").
		Prefix("/*pre*/").
		PrefixExpr(builder.Expr("/*prex*/")).
		ID(1).
		OrderBy("id DESC").
		Limit(10).
		Offset(0).
		Suffix("/*suf*/").
		SuffixExpr(builder.Expr("/*sufx*/"))
}

func TestUpdateWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = Update[*Test](e).
		Prefix("/*pre*/").
		PrefixExpr(builder.Expr("/*prex*/")).
		Set("str", "x").
		SetMap(map[string]any{"str": "y"}).
		Where("id = ?", 1).
		ID(1).
		OrderBy("id").
		Limit(10).
		Offset(0).
		Suffix("/*suf*/").
		SuffixExpr(builder.Expr("/*sufx*/"))
}

func TestDeleteExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Delete(e).From("test").Prefix("INVALID").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Update[*Test](e).Prefix("INVALID").Set("str", "x").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}
