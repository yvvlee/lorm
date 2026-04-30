package lorm

import (
	"strings"

	json "github.com/bytedance/sonic"
	"github.com/samber/lo"
)

// FieldFlag describes special handling for a model field.
type FieldFlag uint8

// HasFlag reports whether f includes flag.
func (f FieldFlag) HasFlag(flag FieldFlag) bool {
	return f&flag == flag
}

// Field flag bits stored in FieldDescriptor.Flag.
const (
	FlagPrimaryKey FieldFlag = 1 << iota
	FlagAutoIncrement
	FlagJson
	FlagCreated
	FlagUpdated
	FlagVersion
)

// FlagTagMap maps field flags to struct tag names.
var FlagTagMap = map[FieldFlag]string{
	FlagPrimaryKey:    "primary_key",
	FlagAutoIncrement: "auto_increment",
	FlagJson:          "json",
	FlagCreated:       "created",
	FlagUpdated:       "updated",
	FlagVersion:       "version",
}

// FileDescriptor describes a source file used by lorm metadata/code generation.
type FileDescriptor struct {
	Path            string
	LormImportAlias string
	Package         string
	Imports         []*Import
	Structs         []*ModelDescriptor
}

// RawVarPrefix returns the stable generated variable prefix for the file.
func (d *FileDescriptor) RawVarPrefix() string {
	return "_lorm_file_" + strings.Replace(strings.TrimSuffix(d.Path, ".go"), "/", "_", -1)
}

// JsonMarshal returns d encoded as JSON.
func (d *FileDescriptor) JsonMarshal() string {
	s, _ := json.MarshalString(d)
	return s
}

// Import describes an imported package reference.
type Import struct {
	Path  string
	Alias string
}

// ModelDescriptor stores struct information
type ModelDescriptor struct {
	Name      string
	TableName string
	Fields    []*FieldDescriptor
}

// FlagFields returns database column names whose flags include flag.
func (m *ModelDescriptor) FlagFields(flag FieldFlag) []string {
	return lo.FilterMap(m.Fields, func(item *FieldDescriptor, _ int) (string, bool) {
		return item.DBField, item.Flag.HasFlag(flag)
	})
}

// AllFields returns all database column names in declaration order.
func (m *ModelDescriptor) AllFields() []string {
	return lo.Map(m.Fields, func(item *FieldDescriptor, _ int) string {
		return item.DBField
	})
}

// FieldDescriptor stores field information
type FieldDescriptor struct {
	Name           string
	FullName       string
	DBField        string
	Type           string
	Flag           FieldFlag
	EnsureFullName string `json:",omitempty"`
	EnsureType     string `json:",omitempty"`
}
