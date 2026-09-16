package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/names"
)

func TestGenerateRejectsColumnAccessorNameConflicts(t *testing.T) {
	for _, tc := range []struct {
		name         string
		declarations string
		fields       string
		want         []string
	}{
		{"embedded", "type Sender struct { Name string }; type Recipient struct { Name string }", "Sender `lorm:\"sender_\"`; Recipient `lorm:\"recipient_\"`", []string{"Sender.Name and Recipient.Name", "Model_Fields.Name"}},
		{"direct_and_embedded", "type Base struct { Name string }", "Name string; Base `lorm:\"base_\"`", []string{"Name and Base.Name", "Model_Fields.Name"}},
		{"all", "", "All bool", []string{"field All", "generated method Model_Fields.All"}},
		{"with_alias", "", "WithAlias string", []string{"field WithAlias", "generated method Model_Fields.WithAlias"}},
		{"embedded_all", "type Base struct { All bool }", "Base", []string{"field Base.All", "generated method Model_Fields.All"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractSource(t, fmt.Sprintf("package validation\nimport \"github.com/yvvlee/lorm\"\n%s\ntype Model struct { lorm.UnimplementedModel; %s }", tc.declarations, tc.fields))
			require.ErrorContains(t, err, "model Model")
			for _, part := range tc.want {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func TestGenerateRejectsFieldsConflictingWithModelMethods(t *testing.T) {
	for _, name := range []string{
		"New", "LormFieldPtr", "LormFieldValue", "LormScan", "LormModelDescriptor", "LormCols",
		"TableName", "LormBeforeInsert", "LormAfterInsert", "LormBeforeUpdate", "LormAfterUpdate",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := extractSource(t, fmt.Sprintf("package validation\nimport \"github.com/yvvlee/lorm\"\ntype Model struct {\nlorm.UnimplementedTable\nID int64 `lorm:\"primary_key,auto_increment\"`\nVersion int64 `lorm:\"version\"`\n%s string\n}", name))
			require.ErrorContains(t, err, "model Model field "+name)
			require.ErrorContains(t, err, "generated method Model."+name)
		})
	}
}

func TestGenerateRejectsEmbeddedFieldsConflictingWithModelMethods(t *testing.T) {
	for _, fields := range []string{"", "private string", "Value string"} {
		_, err := extractSource(t, fmt.Sprintf("package validation\nimport \"github.com/yvvlee/lorm\"\ntype New struct { %s }\ntype Model struct { lorm.UnimplementedModel; *New; ID int64 }", fields))
		require.ErrorContains(t, err, "model Model field New conflicts with generated method Model.New")
	}
}

func TestGenerateNameConflictPreservesExistingOutput(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/conflict\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
	copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
	model := filepath.Join(dir, "model.go")
	writeFile(t, model, "package conflict\nimport \"github.com/yvvlee/lorm\"\ntype Model struct { lorm.UnimplementedModel; All bool }\n")
	outputPath := filepath.Join(dir, "model_lorm_gen.go")
	const original = "package conflict\n// Existing generated file must survive failed validation.\n"
	writeFile(t, outputPath, original)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	err = g.Generate([]string{model})
	require.ErrorContains(t, err, "generated method Model_Fields.All")
	after, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Equal(t, original, string(after))
}

func TestGenerateAllowsNamesInSeparateScopes(t *testing.T) {
	info, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type Base struct {
 New string
 LormScan string
 TableName string
}
type Projection struct {
 lorm.UnimplementedModel
 Base
 LormBeforeInsert string
 LormAfterInsert string
 LormBeforeUpdate string
 LormAfterUpdate string
}
type PlainTable struct {
 lorm.UnimplementedTable
 LormBeforeInsert string
 LormAfterInsert string
 LormBeforeUpdate string
 LormAfterUpdate string
 AllValue string `+"`lorm:\"all\"`"+`
 AliasValue string `+"`lorm:\"with_alias\"`"+`
}
`)
	require.NoError(t, err)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	_, err = g.generateFile(info)
	require.NoError(t, err)
	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = filepath.Dir(info.Path)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
