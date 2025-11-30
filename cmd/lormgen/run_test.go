package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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

func mustWrite(t *testing.T, p, c string) {
	t.Helper()
	err := os.WriteFile(p, []byte(c), 0644)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
}
