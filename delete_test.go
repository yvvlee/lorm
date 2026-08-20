package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestDeleteRequiresWhereUnlessGlobalWriteIsExplicit(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	_, err := engine.Delete[*reservedWordModel]().Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause or AllowGlobalWrite")
	assert.Empty(t, recorder.Calls())

	_, err = engine.Delete[*reservedWordModel]().Where(builder.Eq{}).Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause or AllowGlobalWrite")
	assert.Empty(t, recorder.Calls())

	rows, err := engine.Delete[*reservedWordModel]().AllowGlobalWrite().Exec(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.Equal(t, "DELETE FROM `order`", recorder.Last().query)
}
