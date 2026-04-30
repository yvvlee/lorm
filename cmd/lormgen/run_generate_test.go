package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRunGenerateRecursiveAcrossPackages(t *testing.T) {
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	oldWd := wd
	oldTableMapper := tableMapper
	oldFieldMapper := fieldMapper
	oldTablePrefix := tablePrefix
	oldTableSuffix := tableSuffix
	oldTagKey := tagKey
	oldFileSuffix := fileSuffix
	oldIgnorePatterns := ignorePatterns
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
		wd = oldWd
		tableMapper = oldTableMapper
		fieldMapper = oldFieldMapper
		tablePrefix = oldTablePrefix
		tableSuffix = oldTableSuffix
		tagKey = oldTagKey
		fileSuffix = oldFileSuffix
		ignorePatterns = oldIgnorePatterns
	})

	tmp := t.TempDir()
	repoRoot := findRepoRoot(t, oldCwd)
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/multipkg\n\ngo 1.25.0\n\nrequire github.com/yvvlee/lorm v0.0.0\n\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repoRoot)+"\n")
	copyFile(t, filepath.Join(repoRoot, "go.sum"), filepath.Join(tmp, "go.sum"))
	writeFile(t, filepath.Join(tmp, "base", "audit.go"), `package base

import "time"

type AuditFields struct {
	CreatedAt time.Time `+"`lorm:\"created_at,created\"`"+`
}
`)
	writeFile(t, filepath.Join(tmp, "model", "user.go"), `package model

import (
	"example.com/multipkg/base"
	"github.com/yvvlee/lorm"
)

type User struct {
	lorm.UnimplementedTable `+"`lorm:\"users\"`"+`
	ID int `+"`lorm:\"id,primary_key,auto_increment\"`"+`
	*base.AuditFields `+"`lorm:\"audit_\"`"+`
}
`)

	require.NoError(t, os.Chdir(tmp))
	wd = tmp
	tableMapper = "snake"
	fieldMapper = "snake"
	tablePrefix = ""
	tableSuffix = ""
	tagKey = "lorm"
	fileSuffix = "_lorm_gen"
	ignorePatterns = nil

	require.NoError(t, run([]string{"./..."}))

	generated, err := os.ReadFile(filepath.Join(tmp, "model", "user_lorm_gen.go"))
	require.NoError(t, err)
	content := string(generated)
	assert.Contains(t, content, "if m.AuditFields == nil")
	assert.Contains(t, content, "m.AuditFields = new(base.AuditFields)")
	assert.Contains(t, content, "AuditFields.CreatedAt")
	assert.Contains(t, content, "audit_created_at")
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

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func findRepoRoot(t *testing.T, start string) string {
	t.Helper()
	dir := start
	for {
		modPath := filepath.Join(dir, "go.mod")
		if b, err := os.ReadFile(modPath); err == nil && strings.Contains(string(b), "module github.com/yvvlee/lorm") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %s", start)
		}
		dir = parent
	}
}
