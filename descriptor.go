package lorm

import (
	"strings"

	json "github.com/bytedance/sonic"
	"github.com/samber/lo"
)

type FieldFlag uint8

func (f FieldFlag) HasFlag(flag FieldFlag) bool {
	return f&flag != 0
}

const (
	FlagPrimaryKey FieldFlag = 1 << iota
	FlagAutoIncrement
	FlagJson
	FlagCreated
	FlagUpdated
	FlagVersion
)

var FlagTagMap = map[FieldFlag]string{
	FlagPrimaryKey:    "primary_key",
	FlagAutoIncrement: "auto_increment",
	FlagJson:          "json",
	FlagCreated:       "created",
	FlagUpdated:       "updated",
	FlagVersion:       "version",
}

type FileDescriptor struct {
	Path            string
	LormImportAlias string
	Package         string
	Imports         []*Import
	Structs         []*ModelDescriptor
}

func (d *FileDescriptor) RawVarPrefix() string {
	return "_lorm_file_" + strings.Replace(strings.TrimSuffix(d.Path, ".go"), "/", "_", -1)
}

func (d *FileDescriptor) JsonMarshal() string {
	s, _ := json.MarshalString(d)
	return s
}

type Import struct {
	Path  string
	Alias string
}

// ModelDescriptor 存储结构体信息
type ModelDescriptor struct {
	Name      string
	TableName string
	Fields    []*FieldDescriptor
}

func (m *ModelDescriptor) FlagFields(flag FieldFlag) []string {
	return lo.FilterMap(m.Fields, func(item *FieldDescriptor, _ int) (string, bool) {
		return item.DBField, item.Flag.HasFlag(flag)
	})
}

func (m *ModelDescriptor) AllFields() []string {
	return lo.Map(m.Fields, func(item *FieldDescriptor, _ int) string {
		return item.DBField
	})
}

// FieldDescriptor 存储字段信息
type FieldDescriptor struct {
	Name     string
	FullName string
	DBField  string
	Type     string
	Flag     FieldFlag
}
