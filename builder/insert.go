package builder

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strings"
)

// InsertBuilder builds INSERT-style statements clause by clause.
type InsertBuilder struct {
	prefixes         []Sqlizer
	statementKeyword string
	options          []string
	into             string
	columns          []string
	values           [][]any
	suffixes         []Sqlizer
	selectBuilder    *SelectBuilder
	returning        []string
}

// ToSql renders the INSERT statement and its bound arguments.
func (b *InsertBuilder) ToSql() (sqlStr string, args []any, err error) {
	if len(b.into) == 0 {
		err = errors.New("insert statements must specify a table")
		return
	}
	if len(b.values) == 0 && b.selectBuilder == nil {
		err = errors.New("insert statements must have at least one set of values or select clause")
		return
	}

	sql := &bytes.Buffer{}

	if len(b.prefixes) > 0 {
		args, err = appendToSql(b.prefixes, sql, " ", args)
		if err != nil {
			return
		}

		sql.WriteString(" ")
	}

	if b.statementKeyword == "" {
		sql.WriteString("INSERT ")
	} else {
		sql.WriteString(b.statementKeyword)
		sql.WriteString(" ")
	}

	if len(b.options) > 0 {
		sql.WriteString(strings.Join(b.options, " "))
		sql.WriteString(" ")
	}

	sql.WriteString("INTO ")
	sql.WriteString(b.into)
	sql.WriteString(" ")

	if len(b.columns) > 0 {
		sql.WriteString("(")
		sql.WriteString(strings.Join(b.columns, ","))
		sql.WriteString(") ")
	}

	if b.selectBuilder != nil {
		args, err = b.appendSelectToSQL(sql, args)
	} else {
		args, err = b.appendValuesToSQL(sql, args)
	}
	if err != nil {
		return
	}

	if len(b.suffixes) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(b.suffixes, sql, " ", args)
		if err != nil {
			return
		}
	}

	if len(b.returning) > 0 {
		sql.WriteString(" RETURNING ")
		sql.WriteString(strings.Join(b.returning, ","))
	}

	sqlStr = sql.String()
	return
}

func (b *InsertBuilder) appendValuesToSQL(w io.Writer, args []any) ([]any, error) {
	if len(b.values) == 0 {
		return args, errors.New("values for insert statements are not set")
	}
	_, _ = io.WriteString(w, "VALUES ")

	for r, row := range b.values {
		if r > 0 {
			_, _ = io.WriteString(w, ",")
		}
		_, _ = io.WriteString(w, "(")
		for v, val := range row {
			if v > 0 {
				_, _ = io.WriteString(w, ",")
			}
			if vs, ok := val.(Sqlizer); ok {
				// Sqlizer values are embedded directly so callers can pass expressions like DEFAULT or subqueries.
				vsql, vargs, err := vs.ToSql()
				if err != nil {
					return nil, err
				}
				_, _ = io.WriteString(w, vsql)
				args = append(args, vargs...)
			} else {
				_, _ = io.WriteString(w, "?")
				args = append(args, val)
			}
		}
		_, _ = io.WriteString(w, ")")
	}

	return args, nil
}

func (b *InsertBuilder) appendSelectToSQL(w io.Writer, args []any) ([]any, error) {
	if b.selectBuilder == nil {
		return args, errors.New("select clause for insert statements are not set")
	}

	selectClause, sArgs, err := b.selectBuilder.ToSql()
	if err != nil {
		return args, err
	}

	io.WriteString(w, selectClause)
	args = append(args, sArgs...)

	return args, nil
}

// Prefix adds an expression to the beginning of the query
func (b *InsertBuilder) Prefix(sql string, args ...any) *InsertBuilder {
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b *InsertBuilder) PrefixExpr(expr Sqlizer) *InsertBuilder {
	b.prefixes = append(b.prefixes, expr)
	return b
}

// Options adds keyword options before the INTO clause of the query.
func (b *InsertBuilder) Options(options ...string) *InsertBuilder {
	b.options = append(b.options, options...)
	return b
}

// Into sets the INTO clause of the query.
func (b *InsertBuilder) Into(from string) *InsertBuilder {
	b.into = from
	return b
}

// Columns adds insert columns to the query.
func (b *InsertBuilder) Columns(columns ...string) *InsertBuilder {
	b.columns = columns
	return b
}

// Values adds a single row's values to the query.
func (b *InsertBuilder) Values(values ...any) *InsertBuilder {
	b.values = append(b.values, values)
	return b
}

// Suffix adds an expression to the end of the query
func (b *InsertBuilder) Suffix(sql string, args ...any) *InsertBuilder {
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b *InsertBuilder) SuffixExpr(expr Sqlizer) *InsertBuilder {
	b.suffixes = append(b.suffixes, expr)
	return b
}

// SetMap replaces the current columns and values from a column-to-value map.
func (b *InsertBuilder) SetMap(clauses map[string]any) *InsertBuilder {
	// Keep the columns in a consistent order by sorting the column key string.
	cols := make([]string, 0, len(clauses))
	for col := range clauses {
		cols = append(cols, col)
	}
	sort.Strings(cols)

	vals := make([]any, 0, len(clauses))
	for _, col := range cols {
		vals = append(vals, clauses[col])
	}
	b.columns = cols
	b.values = [][]any{vals}
	return b
}

// Select uses a SELECT statement as the INSERT source.
// If both Values and Select are set, Select takes precedence.
func (b *InsertBuilder) Select(sb *SelectBuilder) *InsertBuilder {
	b.selectBuilder = sb
	return b
}

// StatementKeyword overrides the leading statement keyword, for example "REPLACE".
func (b *InsertBuilder) StatementKeyword(keyword string) *InsertBuilder {
	b.statementKeyword = keyword
	return b
}

// Returning adds a RETURNING clause to the query.
func (b *InsertBuilder) Returning(columns ...string) *InsertBuilder {
	b.returning = append(b.returning, columns...)
	return b
}

// Clear resets all fields while preserving slice capacity for reuse.
func (b *InsertBuilder) Clear() *InsertBuilder {
	b.prefixes = resetSlice(b.prefixes)
	b.statementKeyword = ""
	b.options = resetSlice(b.options)
	b.into = ""
	b.columns = resetSlice(b.columns)
	b.values = resetSlice(b.values)
	b.suffixes = resetSlice(b.suffixes)
	b.selectBuilder = nil
	b.returning = resetSlice(b.returning)
	return b
}
