package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionCloseCommitBranches(t *testing.T) {
	e := &Engine{config: &Config{}}
	s := e.session(context.TODO())

	// proxy when no tx
	p := s.proxy()
	assert.Nil(t, p)

	// close with nil tx
	err := s.close()
	assert.NoError(t, err)

	// close again should be no-op
	err = s.close()
	assert.NoError(t, err)

	// commit when closed should be no-op
	err = s.commit()
	assert.NoError(t, err)
}
