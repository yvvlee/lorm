package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm/names"
)

func TestGenerateRejectsGenericModels(t *testing.T) {
	for _, marker := range []string{"UnimplementedTable", "UnimplementedModel"} {
		_, err := extractSource(t, fmt.Sprintf(`package validation
import "github.com/yvvlee/lorm"
type Model[T any] struct { lorm.%s; ID int64; Value T }
`, marker))
		require.ErrorContains(t, err, "model Model: generic models are not supported")
	}
	info, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type Container[T any] struct { Value T }
type Model struct { lorm.UnimplementedModel; ID int64 }
`)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
}

func TestGenerateRejectsRawBytesFields(t *testing.T) {
	for _, typ := range []string{"sql.RawBytes", "Alias", "*sql.RawBytes", "**Alias", "Pointer"} {
		for _, tag := range []string{"", "`lorm:\"json\"`"} {
			t.Run(typ+tag, func(t *testing.T) {
				_, err := extractSource(t, fmt.Sprintf(`package validation
import (
 "database/sql"
 "github.com/yvvlee/lorm"
)
type Alias = sql.RawBytes
type Pointer *sql.RawBytes
type Base struct { Data %s %s }
type Model struct { lorm.UnimplementedTable; ID int64; *Base }
`, typ, tag))
				require.ErrorContains(t, err, "Model.Base.Data")
				require.ErrorContains(t, err, "sql.RawBytes borrows driver memory; use []byte instead")
			})
		}
	}
	info, err := extractSource(t, `package validation
import (
 db "database/sql"
 "github.com/yvvlee/lorm"
)
type Bytes db.RawBytes
type Model struct {
 lorm.UnimplementedModel
 Data []byte
 Named Bytes
 JSON []byte `+"`lorm:\"json\"`"+`
 private db.RawBytes
}
`)
	require.NoError(t, err)
	require.Equal(t, []string{"data", "named", "json"}, info.Structs[0].AllFields())
}

func TestInvalidResultTypesPreserveGeneratedFile(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	for _, declaration := range []string{
		"type Model[T any] struct { lorm.UnimplementedModel; ID int64 }",
		"type Model struct { lorm.UnimplementedModel; Data sql.RawBytes }",
	} {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/invalidmodel\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
		copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
		model := filepath.Join(dir, "model.go")
		writeFile(t, model, "package invalidmodel\nimport (\"github.com/yvvlee/lorm\"; \"database/sql\")\nvar _ sql.RawBytes\n"+declaration+"\n")
		output := filepath.Join(dir, "model_lorm_gen.go")
		const previous = "package invalidmodel\n// Keep existing output on generation failure.\n"
		writeFile(t, output, previous)
		g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
		require.Error(t, g.Generate([]string{model}))
		content, err := os.ReadFile(output)
		require.NoError(t, err)
		require.Equal(t, previous, string(content))
	}
}
