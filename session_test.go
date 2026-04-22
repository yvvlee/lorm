package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionProxyBranches(t *testing.T) {
	e := &Engine{config: &Config{}}
	s := e.session(context.TODO())

	// proxy when no tx
	p := s.proxy()
	assert.Nil(t, p)
}
