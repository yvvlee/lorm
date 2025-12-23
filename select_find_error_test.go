package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryModelFindErrorBranch(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()
	_, err := Query[*Test](engine).Prefix("INVALID").Find(ctx)
	assert.Error(t, err)
}
