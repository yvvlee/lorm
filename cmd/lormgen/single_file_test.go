package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm/names"
)

func TestGenerateSingleFileResolvesSiblingTypes(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/singlefile\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
	copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
	writeFile(t, filepath.Join(dir, "base.go"), `package singlefile
import "time"
type Version = int64
type Base struct {
 Name string
 Updated time.Time `+"`lorm:\"updated\"`"+`
}
`)
	model := filepath.Join(dir, "model.go")
	writeFile(t, model, `package singlefile
import "github.com/yvvlee/lorm"
type User struct {
 lorm.UnimplementedTable
 ID int64 `+"`lorm:\"primary_key\"`"+`
 Version Version `+"`lorm:\"version\"`"+`
 *Base
}
`)
	// Loading this model for type information must not validate or generate it.
	writeFile(t, filepath.Join(dir, "unselected.go"), "package singlefile\nimport \"github.com/yvvlee/lorm\"\ntype Unselected struct { lorm.UnimplementedModel; All string }\n")
	const preserved = "package singlefile\n// Keep this existing output.\n"
	unselectedOutput := filepath.Join(dir, "unselected_custom.go")
	writeFile(t, unselectedOutput, preserved)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "_custom")
	require.NoError(t, g.Generate([]string{model}))
	require.NoFileExists(t, filepath.Join(dir, "base_custom.go"))
	content, err := os.ReadFile(unselectedOutput)
	require.NoError(t, err)
	require.Equal(t, preserved, string(content))

	writeFile(t, filepath.Join(dir, "model_test.go"), `package singlefile
import (
 "testing"
 "time"
 "github.com/yvvlee/lorm"
)
func TestGeneratedSiblingFields(t *testing.T) {
 model := &User{ID: 7, Version: 2, Base: &Base{Name: "test"}}
 if got := model.LormFieldValue("name"); got != "test" { t.Fatal(got) }
 plan, err := model.LormBeforeUpdate(time.Unix(100, 0))
 if err != nil { t.Fatal(err) }
 if plan.PrimaryKeyCount != 1 || len(plan.Where) != 2 || len(plan.Increment) != 1 { t.Fatal(plan) }
 if model.LormModelDescriptor().Fields[3].Flag != lorm.FlagUpdated { t.Fatal("missing managed time metadata") }
}
`)
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)

	// Repeated generation must continue to exclude generated and test files.
	require.NoError(t, g.Generate([]string{model}))
	require.NoFileExists(t, filepath.Join(dir, "model_custom_custom.go"))
}
