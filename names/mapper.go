package names

import (
	"github.com/samber/lo"
)

// Mapper converts Go identifiers into database table or column names.
type Mapper interface {
	ConvertName(string) string
}

// SameMapper leaves names unchanged.
type SameMapper struct{}

func (m SameMapper) ConvertName(o string) string {
	return o
}

// SnakeMapper converts names to snake_case.
type SnakeMapper struct{}

func (mapper SnakeMapper) ConvertName(name string) string {
	return lo.SnakeCase(name)
}

// CamelMapper converts names to camelCase.
type CamelMapper struct{}

func (mapper CamelMapper) ConvertName(name string) string {
	return lo.CamelCase(name)
}

// PrefixMapper prepends a fixed prefix after delegating to another Mapper.
type PrefixMapper struct {
	Mapper Mapper
	Prefix string
}

func (mapper PrefixMapper) ConvertName(name string) string {
	return mapper.Prefix + mapper.Mapper.ConvertName(name)
}

// NewPrefixMapper wraps mapper so every converted name starts with prefix.
func NewPrefixMapper(mapper Mapper, prefix string) PrefixMapper {
	return PrefixMapper{mapper, prefix}
}

// SuffixMapper appends a fixed suffix after delegating to another Mapper.
type SuffixMapper struct {
	Mapper Mapper
	Suffix string
}

func (mapper SuffixMapper) ConvertName(name string) string {
	return mapper.Mapper.ConvertName(name) + mapper.Suffix
}

// NewSuffixMapper wraps mapper so every converted name ends with suffix.
func NewSuffixMapper(mapper Mapper, suffix string) SuffixMapper {
	return SuffixMapper{mapper, suffix}
}
