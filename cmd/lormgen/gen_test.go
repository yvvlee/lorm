package main

import (
	"embed"
	"os"
	"testing"

	json "github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"

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

	fileInfo := generator.extractFile(pkg.Syntax[0])
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

	fileInfo = generator.extractFile(pkg.Syntax[1])
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
