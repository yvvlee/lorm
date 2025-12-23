package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryModelGetErrorBranch(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	// inject invalid prefix to cause SQL error
	_, err := Query[*Test](e).Prefix("INVALID /*error*/").Limit(1).Get(ctx)
	assert.Error(t, err)
}
