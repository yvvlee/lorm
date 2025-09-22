package names

import (
	"github.com/samber/lo"
)

// Mapper represents a name convertor
// Eg: struct's fields name -> table's column name
// Eg: struct name -> table name
type Mapper interface {
	ConvertName(string) string
}

type SameMapper struct{}

func (m SameMapper) ConvertName(o string) string {
	return o
}

type SnakeMapper struct{}

func (mapper SnakeMapper) ConvertName(name string) string {
	return lo.SnakeCase(name)
}

type CamelMapper struct{}

func (mapper CamelMapper) ConvertName(name string) string {
	return lo.CamelCase(name)
}

type PrefixMapper struct {
	Mapper Mapper
	Prefix string
}

func (mapper PrefixMapper) ConvertName(name string) string {
	return mapper.Prefix + mapper.Mapper.ConvertName(name)
}

func NewPrefixMapper(mapper Mapper, prefix string) PrefixMapper {
	return PrefixMapper{mapper, prefix}
}

type SuffixMapper struct {
	Mapper Mapper
	Suffix string
}

func (mapper SuffixMapper) ConvertName(name string) string {
	return mapper.Mapper.ConvertName(name) + mapper.Suffix
}

func NewSuffixMapper(mapper Mapper, suffix string) SuffixMapper {
	return SuffixMapper{mapper, suffix}
}
