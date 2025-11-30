package lorm

import (
	"testing"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestQueryModelStmtWrappers(t *testing.T) {
	engine := &Engine{config: &Config{escaper: names.NoEscaper}}
	_ = Query[*Test](engine).
		Prefix("/*p*/").
		PrefixExpr(builder.Expr("/*px*/")).
		Distinct().
		Options("opt").
		Select("id").
		AddColumn("id").
		RemoveColumns().
		Column("id").
		From("test").
		FromSelect(builder.Select("id").From("test"), "t").
		JoinClause("id = ?", 1).
		Join("JOIN t2 ON t2.id = test.id").
		LeftJoin("LEFT JOIN t2 ON t2.id = test.id").
		RightJoin("RIGHT JOIN t2 ON t2.id = test.id").
		InnerJoin("INNER JOIN t2 ON t2.id = test.id").
		CrossJoin("CROSS JOIN t2").
		Where("id = ?", 1).
		GroupBy("id").
		Having("COUNT(id) > ?", 0).
		OrderByClause("id > ?", 0).
		OrderBy("id").
		Limit(10).
		RemoveLimit().
		Offset(0).
		RemoveOffset().
		Suffix("--end").
		SuffixExpr(builder.Expr("/*x*/"))
}

func TestQueryColStmtWrappers(t *testing.T) {
	engine := &Engine{config: &Config{escaper: names.NoEscaper}}
	_ = QueryCol[uint64](engine).
		Prefix("/*p*/").
		PrefixExpr(builder.Expr("/*px*/")).
		Distinct().
		Options("opt").
		Columns("id").
		RemoveColumns().
		Column("id").
		From("test").
		FromSelect(builder.Select("id").From("test"), "t").
		JoinClause("id = ?", 1).
		Join("JOIN t2 ON t2.id = test.id").
		LeftJoin("LEFT JOIN t2 ON t2.id = test.id").
		RightJoin("RIGHT JOIN t2 ON t2.id = test.id").
		InnerJoin("INNER JOIN t2 ON t2.id = test.id").
		CrossJoin("CROSS JOIN t2").
		Where("id = ?", 1).
		GroupBy("id").
		Having("COUNT(id) > ?", 0).
		OrderByClause("id > ?", 0).
		OrderBy("id").
		Limit(10).
		RemoveLimit().
		Offset(0).
		RemoveOffset().
		Suffix("--end").
		SuffixExpr(builder.Expr("/*x*/"))
}
