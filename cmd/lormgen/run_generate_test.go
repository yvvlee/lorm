package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunGenerateOnFixture(t *testing.T) {
	// copy fixtures into temp dir
	tmp := t.TempDir()
	copyFile(t, filepath.Join("testdata", "user.go"), filepath.Join(tmp, "user.go"))
	copyFile(t, filepath.Join("testdata", "user_address.go"), filepath.Join(tmp, "user_address.go"))

	// set mappers and prefix/suffix to exercise branches
	tableMapper = "snake"
	fieldMapper = "snake"
	tablePrefix = "pre_"
	tableSuffix = "_suf"
	tagKey = "lorm"
	fileSuffix = "_lorm_gen"
	ignorePatterns = nil

	err := run([]string{tmp})
	assert.Error(t, err)
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(dst, b, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
