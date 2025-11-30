package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm/builder"
	"testing"
)

func TestDeleteWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = Delete(e).
		From("test").
		Prefix("/*pre*/").
		PrefixExpr(builder.Expr("/*prex*/")).
		Where("id = ?", 1).
		ID(1).
		OrderBy("id DESC").
		Limit(10).
		Offset(0).
		Suffix("/*suf*/").
		SuffixExpr(builder.Expr("/*sufx*/"))
}

func TestUpdateWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = Update(e).
		Table("test").
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
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	_, err = Delete(e).From("test").Prefix("INVALID").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecErrorWithInvalidPrefix(t *testing.T) {
	driver := "mysql"
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	e, err := NewEngine(driver, dsn)
	if err != nil {
		t.Skip("db not available")
		return
	}
	defer e.Close()
	_, err = Update(e).Table("test").Prefix("INVALID").Set("str", "x").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}
