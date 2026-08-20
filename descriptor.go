package lorm

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"

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
	Path             string
	LormImportAlias  string
	Package          string
	Imports          []*Import
	Structs          []*ModelDescriptor
	Requires64BitInt bool `json:"-"`
}

// RawVarPrefix returns the stable generated variable prefix for the file.
func (d *FileDescriptor) RawVarPrefix() string {
	path := strings.TrimSuffix(d.Path, ".go")
	var normalized strings.Builder
	for _, r := range path {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			normalized.WriteRune(r)
			continue
		}
		normalized.WriteByte('_')
	}
	digest := sha256.Sum256([]byte(d.Path))
	return fmt.Sprintf("_lorm_file_%s_%x", normalized.String(), digest[:4])
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
	Name        string
	TableName   string
	Fields      []*FieldDescriptor
	PrimaryKeys []string
}

// EnsureFields returns embedded pointer fields that generated code must allocate.
func (m *ModelDescriptor) EnsureFields() []*FieldDescriptor {
	seen := make(map[string]struct{})
	fields := make([]*FieldDescriptor, 0)
	for _, field := range m.Fields {
		if field.EnsureFullName == "" {
			continue
		}
		if _, ok := seen[field.EnsureFullName]; ok {
			continue
		}
		seen[field.EnsureFullName] = struct{}{}
		fields = append(fields, field)
	}
	return fields
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

// NeedsBeforeInsertHook reports whether generated code must initialize managed
// insert-time fields or handle an auto-increment field.
func (m *ModelDescriptor) NeedsBeforeInsertHook() bool {
	return m.hasAnyFieldFlag(FlagAutoIncrement | FlagCreated | FlagUpdated)
}

// NeedsAfterInsertHook reports whether generated code must backfill an ID.
func (m *ModelDescriptor) NeedsAfterInsertHook() bool {
	return m.hasAnyFieldFlag(FlagAutoIncrement)
}

// NeedsBeforeUpdateHook reports whether generated code must prepare managed
// fields, primary-key predicates, or optimistic-lock values.
func (m *ModelDescriptor) NeedsBeforeUpdateHook() bool {
	return m.hasAnyFieldFlag(FlagPrimaryKey | FlagCreated | FlagUpdated | FlagVersion)
}

// NeedsAfterUpdateHook reports whether generated code must apply managed
// update-time or optimistic-lock values.
func (m *ModelDescriptor) NeedsAfterUpdateHook() bool {
	return m.hasAnyFieldFlag(FlagUpdated | FlagVersion)
}

func (m *ModelDescriptor) hasAnyFieldFlag(flags FieldFlag) bool {
	for _, field := range m.Fields {
		if field != nil && field.Flag&flags != 0 {
			return true
		}
	}
	return false
}

// FieldDescriptor stores field information
type FieldDescriptor struct {
	Name            string
	FullName        string
	DBField         string
	Type            string
	Flag            FieldFlag
	Pointer         bool   `json:"-"`
	EnsureFullName  string `json:",omitempty"`
	EnsureType      string `json:",omitempty"`
	ManagedTimeKind string `json:"-"`
	IntegerKind     string `json:"-"`
	IntegerBits     int    `json:"-"`
}
