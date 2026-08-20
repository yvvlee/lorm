package builder

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// UpdateBuilder builds UPDATE statements clause by clause.
type UpdateBuilder struct {
	prefixes   []Sqlizer
	table      string
	setClauses []setClause
	from       Sqlizer
	whereParts []Sqlizer
	orderBys   []string
	limit      string
	offset     string
	suffixes   []Sqlizer
	returning  []string
}

type setClause struct {
	column string
	value  any
}

// Clone returns a shallow copy of the builder and its clause slices.
func (b *UpdateBuilder) Clone() *UpdateBuilder {
	if b == nil {
		return nil
	}
	return &UpdateBuilder{
		prefixes:   cloneSqlizers(b.prefixes),
		table:      b.table,
		setClauses: append([]setClause(nil), b.setClauses...),
		from:       b.from,
		whereParts: cloneSqlizers(b.whereParts),
		orderBys:   cloneStrings(b.orderBys),
		limit:      b.limit,
		offset:     b.offset,
		suffixes:   cloneSqlizers(b.suffixes),
		returning:  cloneStrings(b.returning),
	}
}

// ToSql renders the UPDATE statement and its bound arguments.
func (b *UpdateBuilder) ToSql() (sqlStr string, args []any, err error) {
	if len(b.table) == 0 {
		err = fmt.Errorf("update statements must specify a table")
		return
	}
	if len(b.setClauses) == 0 {
		err = fmt.Errorf("update statements must have at least one Set clause")
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

	sql.WriteString("UPDATE ")
	sql.WriteString(b.table)

	sql.WriteString(" SET ")
	setSqls := make([]string, len(b.setClauses))
	for i, clause := range b.setClauses {
		var valSql string
		if vs, ok := clause.value.(Sqlizer); ok {
			vsql, vargs, err := vs.ToSql()
			if err != nil {
				return "", nil, err
			}
			if _, ok := vs.(*SelectBuilder); ok {
				// Subqueries in SET need parentheses, while other Sqlizers can provide their own syntax.
				valSql = fmt.Sprintf("(%s)", vsql)
			} else {
				valSql = vsql
			}
			args = append(args, vargs...)
		} else {
			valSql = "?"
			args = append(args, clause.value)
		}
		setSqls[i] = fmt.Sprintf("%s = %s", clause.column, valSql)
	}
	sql.WriteString(strings.Join(setSqls, ", "))

	if b.from != nil {
		sql.WriteString(" FROM ")
		args, err = appendToSql([]Sqlizer{b.from}, sql, "", args)
		if err != nil {
			return
		}
	}

	if len(b.whereParts) > 0 {
		sql.WriteString(" WHERE ")
		args, err = appendToSql(b.whereParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(b.orderBys) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(b.orderBys, ", "))
	}

	if len(b.limit) > 0 {
		sql.WriteString(" LIMIT ")
		sql.WriteString(b.limit)
	}

	if len(b.offset) > 0 {
		sql.WriteString(" OFFSET ")
		sql.WriteString(b.offset)
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

// Prefix adds an expression to the beginning of the query
func (b *UpdateBuilder) Prefix(sql string, args ...any) *UpdateBuilder {
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b *UpdateBuilder) PrefixExpr(expr Sqlizer) *UpdateBuilder {
	b.prefixes = append(b.prefixes, expr)
	return b
}

// Table sets the table to be updated.
func (b *UpdateBuilder) Table(table string) *UpdateBuilder {
	b.table = table
	return b
}

// Set adds SET clauses to the query.
func (b *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	b.setClauses = append(b.setClauses, setClause{column, value})
	return b
}

// SetMap is a convenience method which calls .Set for each key/value pair in clauses.
func (b *UpdateBuilder) SetMap(clauses map[string]any) *UpdateBuilder {
	keys := make([]string, len(clauses))
	i := 0
	for key := range clauses {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		val := clauses[key]
		b.Set(key, val)
	}
	return b
}

// From adds a PostgreSQL-style FROM clause to the query.
func (b *UpdateBuilder) From(from string) *UpdateBuilder {
	b.from = newPart(from)
	return b
}

// FromSelect sets a subquery into the FROM clause of the query.
func (b *UpdateBuilder) FromSelect(from *SelectBuilder, alias string) *UpdateBuilder {
	b.from = Alias(from, alias)
	return b
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (b *UpdateBuilder) Where(pred any, args ...any) *UpdateBuilder {
	if pred == nil || pred == "" {
		return b
	}
	b.whereParts = append(b.whereParts, newWherePart(pred, args...))
	return b
}

// HasWhere reports whether the statement contains a restrictive WHERE clause.
func (b *UpdateBuilder) HasWhere() bool {
	return hasEffectiveWhere(b.whereParts)
}

// OrderBy adds ORDER BY expressions to the query.
func (b *UpdateBuilder) OrderBy(orderBys ...string) *UpdateBuilder {
	b.orderBys = append(b.orderBys, orderBys...)
	return b
}

// Limit sets a LIMIT clause on the query.
func (b *UpdateBuilder) Limit(limit uint64) *UpdateBuilder {
	b.limit = strconv.FormatUint(limit, 10)
	return b
}

// Offset sets a OFFSET clause on the query.
func (b *UpdateBuilder) Offset(offset uint64) *UpdateBuilder {
	b.offset = strconv.FormatUint(offset, 10)
	return b
}

// Suffix adds an expression to the end of the query
func (b *UpdateBuilder) Suffix(sql string, args ...any) *UpdateBuilder {
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b *UpdateBuilder) SuffixExpr(expr Sqlizer) *UpdateBuilder {
	b.suffixes = append(b.suffixes, expr)
	return b
}

// Returning adds a RETURNING clause to the query.
func (b *UpdateBuilder) Returning(columns ...string) *UpdateBuilder {
	b.returning = append(b.returning, columns...)
	return b
}

// Clear resets all fields while preserving slice capacity for reuse.
func (b *UpdateBuilder) Clear() *UpdateBuilder {
	b.prefixes = resetSlice(b.prefixes)
	b.table = ""
	b.setClauses = resetSlice(b.setClauses)
	b.from = nil
	b.whereParts = resetSlice(b.whereParts)
	b.orderBys = resetSlice(b.orderBys)
	b.limit = ""
	b.offset = ""
	b.suffixes = resetSlice(b.suffixes)
	b.returning = resetSlice(b.returning)
	return b
}
