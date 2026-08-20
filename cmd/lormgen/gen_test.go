package main

import (
	"embed"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	json "github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"github.com/yvvlee/lorm"
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

func TestGenerateCodeEmitsNilSafeFieldValues(t *testing.T) {
	content, err := generateCode(&lorm.FileDescriptor{
		Path:            "model.go",
		Package:         "models",
		LormImportAlias: "lorm",
		Structs: []*lorm.ModelDescriptor{{
			Name: "Record",
			Fields: []*lorm.FieldDescriptor{{
				Name:     "Amount",
				FullName: "Amount",
				DBField:  "amount",
				Type:     "*decimal.Decimal",
				Pointer:  true,
			}},
		}},
	})
	require.NoError(t, err)
	assert.Contains(t, string(content), "func (m *Record) LormFieldValue(name string) any")
	assert.Contains(t, string(content), "if m.Amount == nil")
	assert.Contains(t, string(content), "return m.Amount")
}

func TestGenerateCodeCachesDescriptorWithPrimaryKeyMetadata(t *testing.T) {
	file := &lorm.FileDescriptor{
		Path:            "model.go",
		Package:         "models",
		LormImportAlias: "lorm",
		Structs: []*lorm.ModelDescriptor{
			{
				Name: "NoPrimary",
				Fields: []*lorm.FieldDescriptor{
					{Name: "Name", FullName: "Name", DBField: "name"},
				},
			},
			{
				Name: "SinglePrimary",
				Fields: []*lorm.FieldDescriptor{
					{Name: "ID", FullName: "ID", DBField: "id", Flag: lorm.FlagPrimaryKey},
				},
			},
			{
				Name: "CompositePrimary",
				Fields: []*lorm.FieldDescriptor{
					{Name: "TenantID", FullName: "TenantID", DBField: "tenant_id", Flag: lorm.FlagPrimaryKey},
					{Name: "ID", FullName: "ID", DBField: "id", Flag: lorm.FlagPrimaryKey},
				},
			},
		},
	}
	content, err := generateCode(file)
	require.NoError(t, err)
	generated := string(content)
	prefix := file.RawVarPrefix()

	assert.Empty(t, file.Structs[0].PrimaryKeys)
	assert.Equal(t, []string{"id"}, file.Structs[1].PrimaryKeys)
	assert.Equal(t, []string{"tenant_id", "id"}, file.Structs[2].PrimaryKeys)
	assert.Contains(t, generated, "var "+prefix+"_SinglePrimary_model_descriptor = "+prefix+"_model_descriptor_map[\"SinglePrimary\"]")
	assert.Contains(t, generated, "return "+prefix+"_SinglePrimary_model_descriptor")
	assert.NotContains(t, generated, "LormPrimaryKey")
}

func TestGenerateCodeEmitsOnlyRelevantWriteHooks(t *testing.T) {
	content, err := generateCode(&lorm.FileDescriptor{
		Path:            "model.go",
		Package:         "models",
		LormImportAlias: "lorm",
		Structs: []*lorm.ModelDescriptor{
			{
				Name:      "Plain",
				TableName: "plain",
				Fields: []*lorm.FieldDescriptor{
					{Name: "Name", FullName: "Name", DBField: "name"},
				},
			},
			{
				Name:      "ManualPrimary",
				TableName: "manual_primary",
				Fields: []*lorm.FieldDescriptor{
					{Name: "ID", FullName: "ID", DBField: "id", Flag: lorm.FlagPrimaryKey},
				},
			},
			{
				Name:      "AutoID",
				TableName: "auto_id",
				Fields: []*lorm.FieldDescriptor{
					{
						Name: "ID", FullName: "ID", DBField: "id",
						Flag:        lorm.FlagPrimaryKey | lorm.FlagAutoIncrement,
						IntegerKind: "int64", IntegerBits: 64,
					},
				},
			},
			{
				Name:      "Created",
				TableName: "created",
				Fields: []*lorm.FieldDescriptor{
					{Name: "CreatedAt", FullName: "CreatedAt", DBField: "created_at", Flag: lorm.FlagCreated, ManagedTimeKind: "int64"},
				},
			},
			{
				Name:      "Versioned",
				TableName: "versioned",
				Fields: []*lorm.FieldDescriptor{
					{Name: "Version", FullName: "Version", DBField: "version", Flag: lorm.FlagVersion},
				},
			},
		},
	})
	require.NoError(t, err)
	generated := string(content)
	assert.Contains(t, generated, "func (m *Plain) LormScan(row lorm.RowScanner) error")

	assert.NotContains(t, generated, "func (m *Plain) LormBeforeInsert")
	assert.NotContains(t, generated, "func (m *Plain) LormAfterInsert")
	assert.NotContains(t, generated, "func (m *Plain) LormBeforeUpdate")
	assert.NotContains(t, generated, "func (m *Plain) LormAfterUpdate")

	assert.NotContains(t, generated, "func (m *ManualPrimary) LormBeforeInsert")
	assert.NotContains(t, generated, "func (m *ManualPrimary) LormAfterInsert")
	assert.Contains(t, generated, "func (m *ManualPrimary) LormBeforeUpdate")
	assert.NotContains(t, generated, "func (m *ManualPrimary) LormAfterUpdate")

	assert.Contains(t, generated, "func (m *AutoID) LormBeforeInsert")
	assert.Contains(t, generated, prefixForGeneratedTest("model.go")+"_AutoID_insert_columns")
	assert.Contains(t, generated, prefixForGeneratedTest("model.go")+"_AutoID_insert_columns_without_auto_increment")
	assert.NotContains(t, generated, "Columns: make([]string")
	assert.Contains(t, generated, "func (m *AutoID) LormAfterInsert")
	assert.Contains(t, generated, "func (m *AutoID) LormBeforeUpdate")
	assert.NotContains(t, generated, "func (m *AutoID) LormAfterUpdate")

	assert.Contains(t, generated, "func (m *Created) LormBeforeInsert")
	assert.NotContains(t, generated, "func (m *Created) LormAfterInsert")
	assert.Contains(t, generated, "func (m *Created) LormBeforeUpdate")
	assert.NotContains(t, generated, "func (m *Created) LormAfterUpdate")

	assert.NotContains(t, generated, "func (m *Versioned) LormBeforeInsert")
	assert.NotContains(t, generated, "func (m *Versioned) LormAfterInsert")
	assert.Contains(t, generated, "func (m *Versioned) LormBeforeUpdate")
	assert.Contains(t, generated, "func (m *Versioned) LormAfterUpdate")
}

func prefixForGeneratedTest(path string) string {
	return (&lorm.FileDescriptor{Path: path}).RawVarPrefix()
}

func TestExtractFileRecognizesPointerAlias(t *testing.T) {
	root, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := findRepoRoot(t, root)
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/pointeralias\n\ngo 1.27\n\ntoolchain go1.27.0\n\nrequire github.com/yvvlee/lorm v0.0.0\n\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repoRoot)+"\n")
	copyFile(t, filepath.Join(repoRoot, "go.sum"), filepath.Join(tmp, "go.sum"))
	modelPath := filepath.Join(tmp, "model.go")
	writeFile(t, modelPath, `package pointeralias

import "github.com/yvvlee/lorm"

type OptionalInt *int

type Result struct {
	lorm.UnimplementedModel
	Value OptionalInt
}
`)

	generator := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "_lorm_gen")
	pkgs, err := generator.load([]string{modelPath})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	file := findSyntaxFile(t, generator, pkgs[0], "model.go")
	info, err := generator.extractFile(pkgs[0], file)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
	require.Len(t, info.Structs[0].Fields, 1)
	assert.True(t, info.Structs[0].Fields[0].Pointer)
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

func TestExtractFileRejectsInvalidModelMetadata(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		errorParts []string
	}{
		{
			name: "unrecognized field tag item",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	Name string ` + "`lorm:\"name,unknown\"`" + `
}
`,
			errorParts: []string{`Invalid.Name`, `unrecognized tag item "unknown"`},
		},
		{
			name: "unrecognized table tag item",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedTable ` + "`lorm:\"invalid,unknown\"`" + `
	ID int
}
`,
			errorParts: []string{`invalid table tag for Invalid`, `unrecognized tag item`},
		},
		{
			name: "duplicate direct database column",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	First  string ` + "`lorm:\"same\"`" + `
	Second string ` + "`lorm:\"same\"`" + `
}
`,
			errorParts: []string{`duplicate database column "same"`, `First and Second`},
		},
		{
			name: "auto increment without primary key",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	ID int64 ` + "`lorm:\"id,auto_increment\"`" + `
}
`,
			errorParts: []string{`Invalid.ID`, `auto_increment requires primary_key`},
		},
		{
			name: "multiple version fields",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	VersionA int64 ` + "`lorm:\"version_a,version\"`" + `
	VersionB int64 ` + "`lorm:\"version_b,version\"`" + `
}
`,
			errorParts: []string{`multiple version fields`, `VersionA and VersionB`},
		},
		{
			name: "unsupported managed time type",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	Created bool ` + "`lorm:\"created_at,created\"`" + `
}
`,
			errorParts: []string{`Invalid.Created`, `created and updated require`, `got bool`},
		},
		{
			name: "narrow managed time integer",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	Created int32 ` + "`lorm:\"created_at,created\"`" + `
}
`,
			errorParts: []string{`Invalid.Created`, `created and updated require`, `got int32`},
		},
		{
			name: "unsupported version type",
			source: `package validation

import "github.com/yvvlee/lorm"

type Version int64

type Invalid struct {
	lorm.UnimplementedModel
	Version Version ` + "`lorm:\"version,version\"`" + `
}
`,
			errorParts: []string{`Invalid.Version`, `version requires a built-in integer type`},
		},
		{
			name: "unsupported auto increment type",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	ID string ` + "`lorm:\"id,primary_key,auto_increment\"`" + `
}
`,
			errorParts: []string{`Invalid.ID`, `auto_increment requires a built-in integer type`},
		},
		{
			name: "conflicting managed flags",
			source: `package validation

import "github.com/yvvlee/lorm"

type Invalid struct {
	lorm.UnimplementedModel
	Value int64 ` + "`lorm:\"value,created,version\"`" + `
}
`,
			errorParts: []string{`Invalid.Value`, `created, updated, and version are mutually exclusive`},
		},
		{
			name: "database column conflict after embedded expansion",
			source: `package validation

import (
	"time"

	"github.com/yvvlee/lorm"
)

type AuditFields struct {
	CreatedAt time.Time ` + "`lorm:\"created_at\"`" + `
}

type Invalid struct {
	lorm.UnimplementedModel
	AuditCreatedAt time.Time ` + "`lorm:\"audit_created_at\"`" + `
	AuditFields ` + "`lorm:\"audit_\"`" + `
}
`,
			errorParts: []string{`duplicate database column "audit_created_at"`, `AuditCreatedAt and AuditFields.CreatedAt`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := extractSource(t, tt.source)
			require.Error(t, err)
			assert.Nil(t, info)
			for _, part := range tt.errorParts {
				assert.ErrorContains(t, err, part)
			}
		})
	}
}

func TestExtractFileAcceptsSupportedManagedTimeTypes(t *testing.T) {
	info, err := extractSource(t, `package validation

import (
	"database/sql"
	"time"

	"github.com/yvvlee/lorm"
)

type Valid struct {
	lorm.UnimplementedModel
	ID          int64         `+"`lorm:\"id,primary_key,auto_increment\"`"+`
	Time        time.Time     `+"`lorm:\"time,created\"`"+`
	TimePtr     *time.Time    `+"`lorm:\"time_ptr,updated\"`"+`
	NullTime    sql.NullTime  `+"`lorm:\"null_time,created\"`"+`
	NullTimePtr *sql.NullTime `+"`lorm:\"null_time_ptr,updated\"`"+`
	Int64       int64         `+"`lorm:\"int64_time,created\"`"+`
	Uint64      uint64        `+"`lorm:\"uint64_time,updated\"`"+`
	Uint32      uint32        `+"`lorm:\"uint32_time,updated\"`"+`
	Uint        uint          `+"`lorm:\"uint_time,created\"`"+`
	Int         int           `+"`lorm:\"int_time,created\"`"+`
	String      string        `+"`lorm:\"string_time,updated\"`"+`
}
`)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
	assert.Len(t, info.Structs[0].Fields, 11)
}

func TestManagedIntTimeTargetWidth(t *testing.T) {
	assert.False(t, supportsManagedIntTime(types.SizesFor("gc", "386")))
	assert.True(t, supportsManagedIntTime(types.SizesFor("gc", "amd64")))
}

func TestExtractFileAcceptsColumnNameAfterFlags(t *testing.T) {
	info, err := extractSource(t, `package validation

import "github.com/yvvlee/lorm"

type Valid struct {
	lorm.UnimplementedModel
	ID int64 `+"`lorm:\"primary_key,id\"`"+`
}
`)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
	require.Len(t, info.Structs[0].Fields, 1)
	assert.Equal(t, "id", info.Structs[0].Fields[0].DBField)
	assert.True(t, info.Structs[0].Fields[0].Flag.HasFlag(lorm.FlagPrimaryKey))
}

func TestExtractFileAcceptsGenericFieldTypes(t *testing.T) {
	info, err := extractSource(t, `package validation

import "github.com/yvvlee/lorm"

type Optional[T any] struct {
	Value T
}

type Pair[A, B any] struct {
	First A
	Second B
}

type Valid struct {
	lorm.UnimplementedModel
	Optional Optional[int]
	Pair Pair[string, int]
}
`)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
	require.Len(t, info.Structs[0].Fields, 2)
	assert.Equal(t, "Optional[int]", info.Structs[0].Fields[0].Type)
	assert.Equal(t, "Pair[string, int]", info.Structs[0].Fields[1].Type)
}

func extractSource(t *testing.T, source string) (*lorm.FileDescriptor, error) {
	t.Helper()

	root, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := findRepoRoot(t, root)
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/validation\n\ngo 1.27\n\ntoolchain go1.27.0\n\nrequire github.com/yvvlee/lorm v0.0.0\n\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repoRoot)+"\n")
	copyFile(t, filepath.Join(repoRoot, "go.sum"), filepath.Join(tmp, "go.sum"))
	modelPath := filepath.Join(tmp, "model.go")
	writeFile(t, modelPath, source)

	generator := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "_lorm_gen")
	pkgs, err := generator.load([]string{modelPath})
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	file := findSyntaxFile(t, generator, pkgs[0], "model.go")
	return generator.extractFile(pkgs[0], file)
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
