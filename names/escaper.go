package names

import (
	"strings"

	"github.com/samber/lo"
)

type Quoter struct {
	prefix byte
	suffix byte
}

func NewQuoter(prefix byte, suffix byte) *Quoter {
	return &Quoter{prefix: prefix, suffix: suffix}
}

func (q Quoter) Escape(fieldOrTable string) string {
	if fieldOrTable == "" {
		return ""
	}
	if q.prefix == 0 && q.suffix == 0 {
		return fieldOrTable
	}
	items := strings.Split(fieldOrTable, ".")
	return strings.Join(lo.Map(items, func(s string, _ int) string {
		s = strings.TrimRight(strings.TrimLeft(s, string(q.prefix)), string(q.suffix))
		return string(q.prefix) + s + string(q.suffix)
	}), ".")
}

type Escaper interface {
	Escape(fieldOrTable string) string
}

var NoEscaper = Escaper(new(Quoter))
