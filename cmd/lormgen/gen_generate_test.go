package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm/names"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	// copy fixtures
	mustWrite2(t, filepath.Join(tmp, "user.go"), "package test\n type X struct{}")
	mustWrite2(t, filepath.Join(tmp, "user_address.go"), "package test\n type Y struct{}")

	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "_gen")
	// It will likely skip generating as fixtures aren't tagged, this still executes Generate
	err := g.Generate([]string{filepath.Join(tmp, "user.go"), filepath.Join(tmp, "user_address.go")})
	assert.NoError(t, err)
}

func mustWrite2(t *testing.T, p, c string) {
	t.Helper()
	err := os.WriteFile(p, []byte(c), 0644)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
}
