package main

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/names"
)

func TestEmptySuffixUsesDefaultAndPreservesSource(t *testing.T) {
	info, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type User struct {
 lorm.UnimplementedTable
 Name string
}
`)
	require.NoError(t, err)
	original, err := os.ReadFile(info.Path)
	require.NoError(t, err)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	path, err := g.generateFile(info)
	require.NoError(t, err)
	assert.Equal(t, "model_lorm_gen.go", filepath.Base(path))
	after, err := os.ReadFile(info.Path)
	require.NoError(t, err)
	assert.Equal(t, original, after)

	previous := fileSuffix
	fileSuffix = ""
	t.Cleanup(func() { fileSuffix = previous })
	assert.True(t, isValidFile(info.Path))
	assert.False(t, isValidFile(path))
}

func TestGenerateRejectsAutoIncrementOnlyTable(t *testing.T) {
	_, err := extractSource(t, "package validation\nimport \"github.com/yvvlee/lorm\"\ntype AutoOnly struct {\nlorm.UnimplementedTable\nID int64 `lorm:\"id,primary_key,auto_increment\"`\n}\n")
	require.ErrorContains(t, err, "AutoOnly contains only auto-increment primary key ID")
}

func TestGeneratedFilePreservesBuildConstraints(t *testing.T) {
	for _, tt := range []struct{ name, header string }{
		{"model.go", ""},
		{"linux.go", ""}, // A bare OS name does not constrain the source file.
		{"model_darwin.go", ""},
		{"model_linux_arm64.go", ""},
		{"model_amd64.go", ""},
		{"model_unix.go", ""}, // unix is an explicit build tag, not a filename suffix.
		{"model.go", "//go:build feature && (linux || darwin)\n\n"},
		{"model_linux.go", "//go:build feature || custom\n\n"},
		{"model.go", "// +build linux darwin\n// +build feature\n\n"},
	} {
		t.Run(tt.name+tt.header, func(t *testing.T) {
			dir := t.TempDir()
			source := tt.header + "package model\n"
			path := filepath.Join(dir, tt.name)
			require.NoError(t, os.WriteFile(path, []byte(source), 0644))
			file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments)
			require.NoError(t, err)
			tags, err := sourceBuildConstraint(path, file)
			require.NoError(t, err)
			g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
			generated, err := g.generateFile(&lorm.FileDescriptor{Path: path, Package: "model", LormImportAlias: "lorm", BuildConstraint: tags})
			require.NoError(t, err)
			for _, goos := range []string{"linux", "android", "darwin", "ios", "windows", "freebsd"} {
				for _, goarch := range []string{"arm64", "amd64", "386"} {
					for _, feature := range []bool{false, true} {
						ctx := build.Default
						ctx.GOOS, ctx.GOARCH = goos, goarch
						ctx.BuildTags = nil
						if feature {
							ctx.BuildTags = []string{"feature"}
						}
						want, err := ctx.MatchFile(dir, tt.name)
						require.NoError(t, err)
						got, err := ctx.MatchFile(dir, filepath.Base(generated))
						require.NoError(t, err)
						assert.Equal(t, want, got, "%s/%s feature=%v", goos, goarch, feature)
					}
				}
			}
		})
	}
}

func TestGeneratePlatformModelCompilesForOtherTarget(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/platform\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
	copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
	writeFile(t, filepath.Join(dir, "common.go"), "package platform\n")
	model := filepath.Join(dir, "model_"+runtime.GOOS+".go")
	writeFile(t, model, "package platform\nimport \"github.com/yvvlee/lorm\"\ntype User struct {\nlorm.UnimplementedTable\nName string\n}\n")
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	require.NoError(t, g.Generate([]string{model}))
	otherOS := "linux"
	if runtime.GOOS == otherOS {
		otherOS = "windows"
	}
	for _, goos := range []string{runtime.GOOS, otherOS} {
		cmd := exec.Command("go", "build", "-mod=mod", "./...")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+runtime.GOARCH)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s: %s", goos, output)
	}
}
