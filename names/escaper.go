package names

import (
	"strings"

	"github.com/samber/lo"
)

// Quoter escapes identifiers with a fixed prefix and suffix.
type Quoter struct {
	prefix byte
	suffix byte
}

// NewQuoter returns a Quoter for the given identifier quote characters.
func NewQuoter(prefix byte, suffix byte) *Quoter {
	return &Quoter{prefix: prefix, suffix: suffix}
}

// Escape quotes each dot-separated identifier part independently.
func (q Quoter) Escape(fieldOrTable string) string {
	if fieldOrTable == "" {
		return ""
	}
	if q.prefix == 0 && q.suffix == 0 {
		return fieldOrTable
	}
	// Quote each segment separately so schema-qualified names stay addressable.
	items := lo.Map(strings.Split(fieldOrTable, "."), func(s string, _ int) string {
		s = strings.TrimPrefix(s, string(q.prefix))
		s = strings.TrimSuffix(s, string(q.suffix))
		if q.suffix != 0 {
			escapedSuffix := string([]byte{q.suffix, q.suffix})
			s = strings.ReplaceAll(s, string(q.suffix), escapedSuffix)
		}
		return string(q.prefix) + s + string(q.suffix)
	})
	return strings.Join(items, ".")
}

// Escaper quotes database identifiers such as columns or tables.
type Escaper interface {
	Escape(fieldOrTable string) string
}

// NoEscaper leaves identifiers unchanged.
var NoEscaper = Escaper(new(Quoter))
