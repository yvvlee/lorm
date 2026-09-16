package names

import (
	"fmt"
	"strings"
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
// Calling Escape on an already-escaped identifier is safe (idempotent).
// Dots inside quoted parts are literal. Malformed quoted identifiers panic.
func (q Quoter) Escape(fieldOrTable string) string {
	if fieldOrTable == "" || (q.prefix == 0 && q.suffix == 0) {
		return fieldOrTable
	}
	if fieldOrTable[0] == q.prefix {
		if q.quotedEnd(fieldOrTable) == len(fieldOrTable) {
			return fieldOrTable
		}
	} else if !strings.Contains(fieldOrTable, ".") {
		// Ordinary column names avoid the qualified-identifier parser.
		return q.escapeSegment(fieldOrTable)
	}
	var result strings.Builder
	result.Grow(len(fieldOrTable) + 2*strings.Count(fieldOrTable, ".") + 2)
	for start := 0; start < len(fieldOrTable); {
		part := fieldOrTable[start:]
		var end int
		if part[0] == q.prefix {
			end = q.quotedEnd(part)
			result.WriteString(part[:end])
		} else {
			end = strings.IndexByte(part, '.')
			if end < 0 {
				end = len(part)
			}
			q.writeSegment(&result, part[:end])
		}
		start += end
		if start < len(fieldOrTable) {
			result.WriteByte('.')
			start++
			if start == len(fieldOrTable) {
				q.writeSegment(&result, "")
			}
		}
	}
	return result.String()
}

// quotedEnd returns the end of one complete quoted part, including its quotes.
func (q Quoter) quotedEnd(s string) int {
	for i := 1; i < len(s); {
		next := strings.IndexByte(s[i:], q.suffix)
		if next < 0 {
			break
		}
		i += next + 1
		if i < len(s) && s[i] == q.suffix {
			i++ // A doubled closing quote is part of the identifier.
			continue
		}
		if i == len(s) || s[i] == '.' {
			return i
		}
		break
	}
	panic(fmt.Sprintf("lorm: malformed quoted identifier %q", s))
}

// escapeSegment quotes a single unquoted identifier segment.
func (q Quoter) escapeSegment(s string) string {
	var result strings.Builder
	result.Grow(len(s) + 2)
	q.writeSegment(&result, s)
	return result.String()
}

func (q Quoter) writeSegment(result *strings.Builder, s string) {
	result.WriteByte(q.prefix)
	for {
		index := strings.IndexByte(s, q.suffix)
		if index < 0 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:index+1])
		result.WriteByte(q.suffix)
		s = s[index+1:]
	}
	result.WriteByte(q.suffix)
}

// Escaper quotes database identifiers such as columns or tables.
type Escaper interface {
	Escape(fieldOrTable string) string
}

// NoEscaper leaves identifiers unchanged.
var NoEscaper = Escaper(new(Quoter))
