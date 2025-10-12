package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm/names"
)

func TestIsValidFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid go file",
			input:    "model.go",
			expected: true,
		},
		{
			name:     "Test file should be ignored",
			input:    "model_test.go",
			expected: false,
		},
		{
			name:     "Generated file should be ignored",
			input:    "model_gen.go",
			expected: false,
		},
		{
			name:     "Generated file with different suffix should be ignored",
			input:    "model_lorm_gen.go",
			expected: false,
		},
		{
			name:     "Non-go file",
			input:    "readme.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFile(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArgsToFiles(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		path    string
		content string
	}{
		{path: "model1.go", content: "package test"},
		{path: "model2.go", content: "package test"},
		{path: "model_test.go", content: "package test"},
		{path: "model_gen.go", content: "package test"},
		{path: "readme.md", content: "# Test"},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tempDir, tf.path)
		err := os.WriteFile(fullPath, []byte(tf.content), 0644)
		assert.NoError(t, err)
	}

	// Test with a single file
	files, err := argsToFiles([]string{filepath.Join(tempDir, "model1.go")})
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	// Test with a directory
	files, err = argsToFiles([]string{tempDir})
	assert.NoError(t, err)
	assert.Len(t, files, 2) // Only .go files that are not tests or generated files

	// Check that we got the right files
	fileNames := make([]string, len(files))
	for i, file := range files {
		fileNames[i] = filepath.Base(file)
	}
	assert.Contains(t, fileNames, "model1.go")
	assert.Contains(t, fileNames, "model2.go")
	assert.NotContains(t, fileNames, "model_test.go")
	assert.NotContains(t, fileNames, "model_gen.go")
	assert.NotContains(t, fileNames, "readme.md")

	// Test with "./..." pattern
	// First create a subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	assert.NoError(t, err)

	subFile := filepath.Join(subDir, "model3.go")
	err = os.WriteFile(subFile, []byte("package test"), 0644)
	assert.NoError(t, err)

	files, err = argsToFiles([]string{tempDir + "/..."})
	assert.NoError(t, err)
	assert.Len(t, files, 3) // model1.go, model2.go and model3.go

	// Check that we got the right files including from subdirectory
	fileNames = make([]string, len(files))
	for i, file := range files {
		fileNames[i] = filepath.Base(file)
	}
	assert.Contains(t, fileNames, "model1.go")
	assert.Contains(t, fileNames, "model2.go")
	assert.Contains(t, fileNames, "model3.go")
}

func TestInitFunctions(t *testing.T) {
	// Test initialization of working directory function
	assert.NotEmpty(t, wd)
	_, err := os.Stat(wd)
	assert.NoError(t, err)

	// Test if mappers are correctly initialized
	assert.NotNil(t, mappers["snake"])
	assert.NotNil(t, mappers["camel"])
	assert.NotNil(t, mappers["same"])
	assert.IsType(t, &names.SnakeMapper{}, mappers["snake"])
	assert.IsType(t, &names.CamelMapper{}, mappers["camel"])
	assert.IsType(t, &names.SameMapper{}, mappers["same"])
}

func TestArgsToFilesWithEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	files, err := argsToFiles([]string{tempDir})
	assert.NoError(t, err)
	assert.Empty(t, files)
}

func TestArgsToFilesWithNonExistentPath(t *testing.T) {
	_, err := argsToFiles([]string{"/non/existent/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestArgsToFilesRecursive(t *testing.T) {
	// Create a test structure with subdirectories
	tempDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	assert.NoError(t, err)

	// Create valid Go files
	validFile := filepath.Join(tempDir, "model.go")
	err = os.WriteFile(validFile, []byte("package model"), 0644)
	assert.NoError(t, err)

	validSubFile := filepath.Join(subDir, "submodel.go")
	err = os.WriteFile(validSubFile, []byte("package model"), 0644)
	assert.NoError(t, err)

	// Create files that should be ignored
	testFile := filepath.Join(tempDir, "model_test.go")
	err = os.WriteFile(testFile, []byte("package model"), 0644)
	assert.NoError(t, err)

	genFile := filepath.Join(tempDir, "model_gen.go")
	err = os.WriteFile(genFile, []byte("package model"), 0644)
	assert.NoError(t, err)

	// Test recursive pattern "./..."
	files, err := argsToFiles([]string{tempDir + "/..."})
	assert.NoError(t, err)
	assert.Len(t, files, 2) // Should only contain two valid files

	// Check the returned file list
	fileBasenames := make([]string, len(files))
	for i, file := range files {
		fileBasenames[i] = filepath.Base(file)
	}
	assert.Contains(t, fileBasenames, "model.go")
	assert.Contains(t, fileBasenames, "submodel.go")
	assert.NotContains(t, fileBasenames, "model_test.go")
	assert.NotContains(t, fileBasenames, "model_gen.go")
}
