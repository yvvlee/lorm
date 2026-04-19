package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
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

func TestDeleteModelWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = DeleteModel[*Test](e).
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

func TestDeleteModelByID(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	_, err := Insert[*Test](e).AddModel(&Test{
		Int:       42,
		Bool:      true,
		Str:       "typed_delete_target",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(4.20),
		IntSlice:  []int{4, 2},
		Struct:    Sub{ID: 42, Name: "typed"},
	}).Exec(ctx)
	assert.NoError(t, err)

	target, err := Query[*Test](e).Where("str = ?", "typed_delete_target").Get(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, target)

	rowsAffected, err := DeleteModel[*Test](e).ID(target.ID).Exec(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	deleted, err := Query[*Test](e).Where("id = ?", target.ID).Get(ctx)
	assert.NoError(t, err)
	assert.Nil(t, deleted)
}
