package lorm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRejectsNonPointerModelTypesAtCompileTime(t *testing.T) {
	tests := map[string]string{
		"scalar": `package compilecheck

import "github.com/yvvlee/lorm"

func invalid(engine *lorm.Engine) {
	engine.Query[int64]()
}
`,
		"modelValue": `package compilecheck

import "github.com/yvvlee/lorm"

type ModelValue struct {
	lorm.UnimplementedModel
}

func (ModelValue) New() lorm.Model { return ModelValue{} }
func (ModelValue) LormFieldPtr(string) any { return nil }
func (ModelValue) LormModelDescriptor() *lorm.ModelDescriptor { return new(lorm.ModelDescriptor) }

func invalid(engine *lorm.Engine) {
	engine.Query[ModelValue]()
}
`,
	}

	root, err := os.Getwd()
	require.NoError(t, err)
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			goMod := "module compilecheck\n\ngo 1.27\n\nrequire github.com/yvvlee/lorm v0.0.0\n\nreplace github.com/yvvlee/lorm => " + filepath.ToSlash(root) + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "query_test.go"), []byte(source), 0o600))

			cmd := exec.Command("go", "test", "-mod=mod", ".")
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "invalid Query type unexpectedly compiled")
			assert.Contains(t, string(output), "does not satisfy lorm.ModelPointer")
		})
	}
}
