package main

import (
	"os"
	"os/exec"
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
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(tmp, "user_lorm_gen.go"))
	assert.FileExists(t, filepath.Join(tmp, "user_address_lorm_gen.go"))
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
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/multipkg\n\ngo 1.27\n\ntoolchain go1.27.0\n\nrequire github.com/yvvlee/lorm v0.0.0\n\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repoRoot)+"\n")
	copyFile(t, filepath.Join(repoRoot, "go.sum"), filepath.Join(tmp, "go.sum"))
	writeFile(t, filepath.Join(tmp, "base", "audit.go"), `package base

import (
	"database/sql"
	"time"
)

type AuditFields struct {
	CreatedAt   time.Time     `+"`lorm:\"created_at,created\"`"+`
	UpdatedAt   *time.Time    `+"`lorm:\"updated_at,updated\"`"+`
	NullTime    sql.NullTime  `+"`lorm:\"null_time,created\"`"+`
	NullTimePtr *sql.NullTime `+"`lorm:\"null_time_ptr,updated\"`"+`
	Int64Time   int64         `+"`lorm:\"int64_time,created\"`"+`
	Uint64Time  *uint64       `+"`lorm:\"uint64_time,updated\"`"+`
	Uint32Time  uint32        `+"`lorm:\"uint32_time,created\"`"+`
	UintTime    *uint         `+"`lorm:\"uint_time,updated\"`"+`
	IntTime     int           `+"`lorm:\"int_time,created\"`"+`
	StringTime  *string       `+"`lorm:\"string_time,updated\"`"+`
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
	Name string `+"`lorm:\"name\"`"+`
	Version *int64 `+"`lorm:\"version,version\"`"+`
	*base.AuditFields `+"`lorm:\"audit_\"`"+`
}
`)
	writeFile(t, filepath.Join(tmp, "model", "usage.go"), `package model

import (
	"testing"
	"time"

	"example.com/multipkg/base"
)

func TestGeneratedWriteHooks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	user := &User{}
	plan := user.LormBeforeInsert(now)
	if user.AuditFields == nil || user.CreatedAt != now || user.UpdatedAt == nil || *user.UpdatedAt != now {
		t.Fatal("managed insert times were not populated")
	}
	if !user.NullTime.Valid || user.NullTime.Time != now ||
		user.NullTimePtr == nil || !user.NullTimePtr.Valid || user.NullTimePtr.Time != now ||
		user.Int64Time != now.Unix() ||
		user.Uint64Time == nil || *user.Uint64Time != uint64(now.Unix()) ||
		user.Uint32Time != uint32(now.Unix()) ||
		user.UintTime == nil || *user.UintTime != uint(now.Unix()) ||
		user.IntTime != int(now.Unix()) ||
		user.StringTime == nil || *user.StringTime != now.Format(time.DateTime) {
		t.Fatal("managed insert time conversions were not populated")
	}
	if plan.AutoIncrementColumn != "id" || !plan.AutoIncrementZero {
		t.Fatalf("unexpected insert plan: %#v", plan)
	}
	preset := &User{AuditFields: &base.AuditFields{Int64Time: 42}}
	preset.LormBeforeInsert(now)
	if preset.Int64Time != 42 {
		t.Fatal("non-zero managed insert time was overwritten")
	}
	if _, err := user.LormBeforeUpdate(now); err == nil {
		t.Fatal("nil version must fail")
	}
	version := int64(3)
	user.Version = &version
	update, err := user.LormBeforeUpdate(now)
	if err != nil || update.PrimaryKeyCount != 1 || len(update.Increment) != 1 {
		t.Fatalf("unexpected update plan: %#v, %v", update, err)
	}
	later := now.Add(time.Hour)
	user.LormAfterUpdate(later, 0)
	if *user.Version != 3 || *user.UpdatedAt != now {
		t.Fatal("zero-row update changed the model")
	}
	user.LormAfterUpdate(later, 1)
	if *user.Version != 4 {
		t.Fatalf("version was not incremented: %d", *user.Version)
	}
	if user.UpdatedAt == nil || *user.UpdatedAt != later ||
		user.NullTimePtr == nil || user.NullTimePtr.Time != later ||
		user.Uint64Time == nil || *user.Uint64Time != uint64(later.Unix()) ||
		user.UintTime == nil || *user.UintTime != uint(later.Unix()) ||
		user.StringTime == nil || *user.StringTime != later.Format(time.DateTime) {
		t.Fatal("managed update time conversions were not applied")
	}
	if user.CreatedAt != now || user.Int64Time != now.Unix() {
		t.Fatal("created fields changed during update")
	}
	_ = user.Fields()
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
	assert.Contains(t, content, "LormBeforeInsert")
	assert.Contains(t, content, "LormBeforeUpdate")
	assert.Contains(t, content, "NilVersionError")
	assert.Contains(t, content, "_requires_64_bit_int")

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = tmp
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
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
