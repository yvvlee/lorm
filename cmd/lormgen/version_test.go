package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrentVersionIsSet(t *testing.T) {
	assert.NotEmpty(t, currentVersion())
	assert.NotEqual(t, "(devel)", currentVersion())
}
