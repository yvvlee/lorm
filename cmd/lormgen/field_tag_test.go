package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm"
)

func TestExtractRejectsConflictingFieldTags(t *testing.T) {
	for _, tc := range []struct{ tag, message string }{
		{"id,primary_key,version", "primary_key cannot be combined"},
		{"id,primary_key,updated", "primary_key cannot be combined"},
		{"id,primary_key,auto_increment,created", "auto_increment cannot be combined"},
		{"id,primary_key,auto_increment,json", "json cannot be combined"},
		{"id,json,version", "json cannot be combined"},
		{"id,json,created", "json cannot be combined"},
		{"id,json,updated", "json cannot be combined"},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			_, err := extractSource(t, fmt.Sprintf("package validation\nimport \"github.com/yvvlee/lorm\"\ntype Model struct {\nlorm.UnimplementedTable\nID int64 `lorm:%q`\nName string\n}\n", tc.tag))
			require.ErrorContains(t, err, "Model.ID")
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestVersionRequiresAtLeast32Bits(t *testing.T) {
	for _, typ := range []string{"int8", "uint8", "int16", "uint16", "byte", "Narrow"} {
		for _, prefix := range []string{"", "*"} {
			t.Run(prefix+typ, func(t *testing.T) {
				_, err := extractSource(t, fmt.Sprintf("package validation\nimport \"github.com/yvvlee/lorm\"\ntype Narrow = int16\ntype Model struct {\nlorm.UnimplementedTable\nID int64 `lorm:\"primary_key\"`\nVersion %s `lorm:\"version\"`\n}\n", prefix+typ))
				require.ErrorContains(t, err, "Model.Version")
				require.ErrorContains(t, err, "at least 32 bits")
			})
		}
	}

	var source strings.Builder
	source.WriteString("package validation\nimport \"github.com/yvvlee/lorm\"\ntype Counter = int32\n")
	for i, typ := range []string{"int32", "uint32", "int64", "uint64", "int", "uint", "rune", "Counter"} {
		for j, prefix := range []string{"", "*"} {
			fmt.Fprintf(&source, "type Model%d_%d struct {\nlorm.UnimplementedTable\nID int64 `lorm:\"primary_key\"`\nVersion %s `lorm:\"version\"`\n}\n", i, j, prefix+typ)
		}
	}
	info, err := extractSource(t, source.String())
	require.NoError(t, err)
	require.Len(t, info.Structs, 16)
}

func TestGenerationSkipsPrivateFieldsBeforeValidation(t *testing.T) {
	info, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type hidden struct { HiddenPublic string }
type hiddenGeneric[T any] struct { GenericPublic T }
type PublicBase struct {
 Exported string
 private func()
}
type Model struct {
 lorm.UnimplementedTable
 hidden
 *hiddenGeneric[int]
 PublicBase
 Name, privateName string
 _ int
 callback func()
 privateVersion int8 `+"`lorm:\"version,unknown\"`"+`
 privateA, privateB int `+"`lorm:\"same\"`"+`
}
`)
	require.NoError(t, err)
	require.Equal(t, []string{"exported", "name"}, info.Structs[0].AllFields())
}

func TestExtractRejectsTaggedGroupedFields(t *testing.T) {
	for _, tag := range []string{"value", "value,version", ""} {
		for _, literal := range []string{"`lorm:" + strconv.Quote(tag) + "`", strconv.Quote("lorm:" + strconv.Quote(tag))} {
			_, err := extractSource(t, "package validation\nimport \"github.com/yvvlee/lorm\"\ntype Model struct {\nlorm.UnimplementedModel\nA, B int64 "+literal+"\n}\n")
			require.ErrorContains(t, err, "grouped fields with a lorm tag must be declared separately")
		}
	}
	info, err := extractSource(t, "package validation\nimport \"github.com/yvvlee/lorm\"\ntype Model struct {\nlorm.UnimplementedModel\nA, B int64\n}\n")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, info.Structs[0].AllFields())
}

func TestExtractInterpretedStructTags(t *testing.T) {
	info, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type Base struct { Name string "lorm:\"display_name\"" }
type User struct {
 lorm.UnimplementedTable "lorm:\"custom_users\""
 ID int64 "lorm:\"user_id,primary_key,auto_increment\""
 Version int64 "lorm:\"revision,version\""
 Base "lorm:\"profile_\""
}
`)
	require.NoError(t, err)
	require.Len(t, info.Structs, 1)
	model := info.Structs[0]
	assert.Equal(t, "custom_users", model.TableName)
	assert.Equal(t, []string{"user_id"}, model.PrimaryKeys)
	assert.Equal(t, []string{"user_id", "revision", "profile_display_name"}, model.AllFields())
	assert.Equal(t, lorm.FlagPrimaryKey|lorm.FlagAutoIncrement, model.Fields[0].Flag)
	assert.Equal(t, lorm.FlagVersion, model.Fields[1].Flag)
}

func TestInterpretedStructTagsPreserveValidation(t *testing.T) {
	_, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type User struct {
 lorm.UnimplementedTable
 ID int64 "lorm:\"id,primary_key,version\""
}
`)
	require.ErrorContains(t, err, "primary_key cannot be combined")
	field := &ast.Field{Tag: &ast.BasicLit{Kind: token.STRING, Value: `"unterminated`}}
	_, _, err = parseFieldTag(field, "lorm")
	require.ErrorContains(t, err, "invalid struct tag literal")
	_, err = parseNameTag(field, "lorm")
	require.ErrorContains(t, err, "invalid struct tag literal")
}
