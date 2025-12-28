package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunArgsAndIgnores(t *testing.T) {
	// no args
	err := run([]string{})
	assert.Error(t, err)

	// unsupported mappers
	tableMapper = "bad"
	err = run([]string{"."})
	assert.Error(t, err)

	tableMapper = "snake"
	fieldMapper = "bad"
	err = run([]string{"."})
	assert.Error(t, err)

	// empty directory should trigger no matching files
	fieldMapper = "snake"
	tempDir := t.TempDir()
	err = run([]string{tempDir})
	assert.Error(t, err)
}
