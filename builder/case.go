package builder

import (
	"bytes"
	"errors"
)

// sqlizerBuffer is a helper that allows to write many Sqlizers one by one
// without constant checks for errors that may come from Sqlizer
type sqlizerBuffer struct {
	bytes.Buffer
	args []any
	err  error
}

// WriteSql converts Sqlizer to SQL strings and writes it to buffer
func (b *sqlizerBuffer) WriteSql(item Sqlizer) {
	if b.err != nil {
		return
	}

	var str string
	var args []any
	str, args, b.err = item.ToSql()

	if b.err != nil {
		return
	}

	b.WriteString(str)
	b.WriteByte(' ')
	b.args = append(b.args, args...)
}

func (b *sqlizerBuffer) ToSql() (string, []any, error) {
	return b.String(), b.args, b.err
}

// whenPart is a helper structure to describe SQLs "WHEN ... THEN ..." expression
type whenPart struct {
	when Sqlizer
	then Sqlizer
}

func newWhenPart(when any, then any) whenPart {
	return whenPart{newPart(when), newPart(then)}
}

// CaseBuilder holds all the data required to build a CASE SQL construct
type CaseBuilder struct {
	what      Sqlizer
	whenParts []whenPart
	els       Sqlizer //else
}

// ToSql implements Sqlizer
func (b *CaseBuilder) ToSql() (sqlStr string, args []any, err error) {
	if len(b.whenParts) == 0 {
		err = errors.New("case expression must contain at lease one WHEN clause")

		return
	}

	sql := sqlizerBuffer{}

	sql.WriteString("CASE ")
	if b.what != nil {
		sql.WriteSql(b.what)
	}

	for _, p := range b.whenParts {
		sql.WriteString("WHEN ")
		sql.WriteSql(p.when)
		sql.WriteString("THEN ")
		sql.WriteSql(p.then)
	}

	if b.els != nil {
		sql.WriteString("ELSE ")
		sql.WriteSql(b.els)
	}

	sql.WriteString("END")

	return sql.ToSql()
}

// What sets optional value for CASE construct "CASE [value] ..."
func (b *CaseBuilder) What(expr any) *CaseBuilder {
	b.what = newPart(expr)
	return b
}

// When adds "WHEN ... THEN ..." part to CASE construct
func (b *CaseBuilder) When(when any, then any) *CaseBuilder {
	b.whenParts = append(b.whenParts, newWhenPart(when, then))
	return b
}

// what sets optional "ELSE ..." part for CASE construct
func (b *CaseBuilder) Else(expr any) *CaseBuilder {
	b.els = newPart(expr)
	return b
}
