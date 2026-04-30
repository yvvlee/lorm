package main

import (
	"embed"
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	json "github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"github.com/yvvlee/lorm/names"
)

//go:embed testdata
var testdata embed.FS

func Test_Generate(t *testing.T) {
	generator := NewGenerator(
		new(names.SnakeMapper),
		new(names.SnakeMapper),
		"lorm",
		"_test_gen",
	)
	pkgs, err := generator.load([]string{
		"testdata/user.go",
		"testdata/user_address.go",
	})
	assert.Nil(t, err)
	assert.Len(t, pkgs, 1)
	pkg := pkgs[0]
	assert.Len(t, pkg.Syntax, 2)

	userFile := findSyntaxFile(t, generator, pkg, "user.go")
	fileInfo, err := generator.extractFile(pkg, userFile)
	require.NoError(t, err)
	fileInfoJson, err := json.MarshalString(fileInfo)
	assert.Nil(t, err)
	assert.NotNil(t, fileInfo)
	exceptFileInfoJson, err := testdata.ReadFile("testdata/user_file_descriptor.json")
	assert.Nil(t, err)
	assert.Equal(t, string(exceptFileInfoJson), fileInfoJson)
	newFile1, err := generator.generateFile(fileInfo)
	assert.Nil(t, err)
	defer os.Remove(newFile1)
	content, err := os.ReadFile(newFile1)
	assert.Nil(t, err)
	exceptContent, err := testdata.ReadFile("testdata/user_lorm_gen.go")
	assert.Nil(t, err)
	assert.Equal(t, string(exceptContent), string(content))

	userAddressFile := findSyntaxFile(t, generator, pkg, "user_address.go")
	fileInfo, err = generator.extractFile(pkg, userAddressFile)
	require.NoError(t, err)
	fileInfoJson, err = json.MarshalString(fileInfo)
	assert.Nil(t, err)
	assert.NotNil(t, fileInfo)
	exceptFileInfoJson, err = testdata.ReadFile("testdata/user_address_file_descriptor.json")
	assert.Nil(t, err)
	assert.Equal(t, string(exceptFileInfoJson), fileInfoJson)
	newFile2, err := generator.generateFile(fileInfo)
	assert.Nil(t, err)
	defer os.Remove(newFile2)
	content, err = os.ReadFile(newFile2)
	assert.Nil(t, err)
	exceptContent, err = testdata.ReadFile("testdata/user_address_lorm_gen.go")
	assert.Nil(t, err)
	assert.Equal(t, string(exceptContent), string(content))
}

func Test_Generate_FlattensEmbeddedStructsAcrossFiles(t *testing.T) {
	generator := NewGenerator(
		new(names.SnakeMapper),
		new(names.SnakeMapper),
		"lorm",
		"_test_gen",
	)
	pkgs, err := generator.load([]string{
		"testdata/audit_base.go",
		"testdata/audit_user.go",
	})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	file := findSyntaxFile(t, generator, pkgs[0], "audit_user.go")
	fileInfo, err := generator.extractFile(pkgs[0], file)
	require.NoError(t, err)
	require.NotNil(t, fileInfo)
	require.Len(t, fileInfo.Structs, 1)

	fields := fileInfo.Structs[0].Fields
	require.Len(t, fields, 3)
	assert.Equal(t, "ID", fields[0].Name)
	assert.Equal(t, "CreatedAt", fields[1].Name)
	assert.Equal(t, "AuditFields.CreatedAt", fields[1].FullName)
	assert.Equal(t, "created_at", fields[1].DBField)
	assert.Equal(t, "UpdatedAt", fields[2].Name)
	assert.Equal(t, "AuditFields.UpdatedAt", fields[2].FullName)
	assert.Equal(t, "updated_at", fields[2].DBField)
}

func Test_Generate_FailsFastForUnsupportedFieldType(t *testing.T) {
	generator := NewGenerator(
		new(names.SnakeMapper),
		new(names.SnakeMapper),
		"lorm",
		"_test_gen",
	)
	pkgs, err := generator.load([]string{
		"testdata/unsupported_type.go",
	})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	file := findSyntaxFile(t, generator, pkgs[0], "unsupported_type.go")
	fileInfo, err := generator.extractFile(pkgs[0], file)
	require.Error(t, err)
	assert.Nil(t, fileInfo)
	assert.ErrorContains(t, err, `unsupported field type "func()"`)
	assert.ErrorContains(t, err, "UnsupportedType.Callback")
}

func findSyntaxFile(t *testing.T, generator *Generator, pkg *packages.Package, name string) *ast.File {
	t.Helper()

	for _, file := range pkg.Syntax {
		tokenFile := generator.fileSet.File(file.Pos())
		if tokenFile == nil {
			continue
		}
		if filepath.Base(tokenFile.Name()) == name {
			return file
		}
	}
	t.Fatalf("syntax file %q not found", name)
	return nil
}
