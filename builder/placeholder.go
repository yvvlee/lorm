package builder

import (
	"bytes"
	"strconv"
	"strings"
)

// PlaceholderFormat rewrites question-mark placeholders into a dialect-specific form.
type PlaceholderFormat interface {
	ReplacePlaceholders(sql string) (string, error)
	PlaceholderString() string
}

var (
	// Question is a PlaceholderFormat instance that leaves placeholders as
	// question marks.
	Question = questionFormat{}

	// Dollar is a PlaceholderFormat instance that replaces placeholders with
	// dollar-prefixed positional placeholders (e.g. $1, $2, $3).
	Dollar = dollarFormat{}

	// Colon is a PlaceholderFormat instance that replaces placeholders with
	// colon-prefixed positional placeholders (e.g. :1, :2, :3).
	Colon = colonFormat{}

	// AtP is a PlaceholderFormat instance that replaces placeholders with
	// "@p"-prefixed positional placeholders (e.g. @p1, @p2, @p3).
	AtP = atpFormat{}
)

type questionFormat struct{}

func (questionFormat) ReplacePlaceholders(sql string) (string, error) {
	return sql, nil
}

func (questionFormat) PlaceholderString() string {
	return "?"
}

type dollarFormat struct{}

func (dollarFormat) ReplacePlaceholders(sql string) (string, error) {
	return replacePositionalPlaceholders(sql, "$")
}

func (dollarFormat) PlaceholderString() string {
	return "$"
}

type colonFormat struct{}

func (colonFormat) ReplacePlaceholders(sql string) (string, error) {
	return replacePositionalPlaceholders(sql, ":")
}

func (colonFormat) PlaceholderString() string {
	return ":"
}

type atpFormat struct{}

func (atpFormat) ReplacePlaceholders(sql string) (string, error) {
	return replacePositionalPlaceholders(sql, "@p")
}

func (atpFormat) PlaceholderString() string {
	return "@p"
}

// Placeholders returns a string with count ? placeholders joined with commas.
func Placeholders(count int) string {
	if count < 1 {
		return ""
	}

	return strings.Repeat(",?", count)[1:]
}

func replacePositionalPlaceholders(sql, prefix string) (string, error) {
	buf := &bytes.Buffer{}
	i := 0
	for {
		p := strings.Index(sql, "?")
		if p == -1 {
			break
		}

		// "??" escapes a literal question mark, so only a single "?" is copied through.
		if len(sql[p:]) > 1 && sql[p:p+2] == "??" {
			buf.WriteString(sql[:p])
			buf.WriteString("?")
			if len(sql[p:]) == 1 {
				break
			}
			sql = sql[p+2:]
		} else {
			i++
			buf.WriteString(sql[:p])
			buf.WriteString(prefix)
			buf.WriteString(strconv.Itoa(i))
			sql = sql[p+1:]
		}
	}

	buf.WriteString(sql)
	return buf.String(), nil
}
