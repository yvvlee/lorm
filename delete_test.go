package lorm

import (
	"testing"

	"github.com/yvvlee/lorm/builder"
)

func TestDeleteWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = e.Delete[*Test]().
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
