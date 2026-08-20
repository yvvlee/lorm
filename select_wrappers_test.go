package lorm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestSelectStmtModelWrappers(t *testing.T) {
	engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: names.NoEscaper}}}
	_ = engine.Query[*Test]().
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

func TestSelectStmtColumnWrappers(t *testing.T) {
	engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: names.NoEscaper}}}
	_ = engine.Query[*Test]().
		Prefix("/*p*/").
		PrefixExpr(builder.Expr("/*px*/")).
		Distinct().
		Options("opt").
		Select("id").
		AddColumn("COUNT(1) AS total").
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

func TestSelectStatementsUseIndependentBuilders(t *testing.T) {
	engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: names.NoEscaper}}}

	q1 := engine.Query[*Test]().Where("id = ?", 1)
	q2 := engine.Query[*Test]().Where("id = ?", 2)

	assert.NotSame(t, q1.builder, q2.builder)

	_, _, err := q1.builder.ToSql()
	assert.ErrorContains(t, err, "at least one result column")
	_, _, err = q2.builder.ToSql()
	assert.ErrorContains(t, err, "at least one result column")

	sql1, args1, err := q1.Select("id").builder.ToSql()
	assert.NoError(t, err)
	sql2, args2, err := q2.Select("index").builder.ToSql()
	assert.NoError(t, err)

	assert.Equal(t, "SELECT id FROM test WHERE id = ?", sql1)
	assert.Equal(t, "SELECT index FROM test WHERE id = ?", sql2)
	assert.Equal(t, []any{1}, args1)
	assert.Equal(t, []any{2}, args2)
}
