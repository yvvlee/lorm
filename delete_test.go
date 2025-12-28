package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
)

func TestDeleteExecError_NoFrom(t *testing.T) {
	e := initEngine(t)
	_, err := Delete(e).Where("id = ?", 1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestDeleteExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Delete(e).From("test").Prefix("INVALID").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

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
