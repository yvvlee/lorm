package builder

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func cloneSqlizers(items []Sqlizer) []Sqlizer {
	return append([]Sqlizer(nil), items...)
}

func cloneStrings(items []string) []string {
	return append([]string(nil), items...)
}

func hasOption(options []string, target string) bool {
	for _, opt := range options {
		if strings.EqualFold(opt, target) {
			return true
		}
	}
	return false
}

// SelectBuilder builds SELECT statements clause by clause.
type SelectBuilder struct {
	withParts    []Sqlizer
	prefixes     []Sqlizer
	options      []string
	columns      []Sqlizer
	from         Sqlizer
	joins        []Sqlizer
	whereParts   []Sqlizer
	groupBys     []string
	havingParts  []Sqlizer
	orderByParts []Sqlizer
	limit        string
	offset       string
	suffixes     []Sqlizer
	lockParts    []Sqlizer
}

// Clone returns a shallow copy of the builder and its clause slices.
func (b *SelectBuilder) Clone() *SelectBuilder {
	if b == nil {
		return nil
	}
	return &SelectBuilder{
		withParts:    cloneSqlizers(b.withParts),
		prefixes:     cloneSqlizers(b.prefixes),
		options:      cloneStrings(b.options),
		columns:      cloneSqlizers(b.columns),
		from:         b.from,
		joins:        cloneSqlizers(b.joins),
		whereParts:   cloneSqlizers(b.whereParts),
		groupBys:     cloneStrings(b.groupBys),
		havingParts:  cloneSqlizers(b.havingParts),
		orderByParts: cloneSqlizers(b.orderByParts),
		limit:        b.limit,
		offset:       b.offset,
		suffixes:     cloneSqlizers(b.suffixes),
		lockParts:    cloneSqlizers(b.lockParts),
	}
}

// ToSql renders the SELECT statement and its bound arguments.
func (b *SelectBuilder) ToSql() (sqlStr string, args []any, err error) {
	if len(b.columns) == 0 {
		err = fmt.Errorf("select statements must have at least one result column")
		return
	}

	sql := &bytes.Buffer{}

	if len(b.withParts) > 0 {
		args, err = appendToSql(b.withParts, sql, " ", args)
		if err != nil {
			return
		}
		sql.WriteString(" ")
	}

	if len(b.prefixes) > 0 {
		args, err = appendToSql(b.prefixes, sql, " ", args)
		if err != nil {
			return
		}

		sql.WriteString(" ")
	}

	sql.WriteString("SELECT ")

	if len(b.options) > 0 {
		sql.WriteString(strings.Join(b.options, " "))
		sql.WriteString(" ")
	}

	if len(b.columns) > 0 {
		args, err = appendToSql(b.columns, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if b.from != nil {
		sql.WriteString(" FROM ")
		args, err = appendToSql([]Sqlizer{b.from}, sql, "", args)
		if err != nil {
			return
		}
	}

	if len(b.joins) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(b.joins, sql, " ", args)
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

	if len(b.groupBys) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(b.groupBys, ", "))
	}

	if len(b.havingParts) > 0 {
		sql.WriteString(" HAVING ")
		args, err = appendToSql(b.havingParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(b.orderByParts) > 0 {
		sql.WriteString(" ORDER BY ")
		args, err = appendToSql(b.orderByParts, sql, ", ", args)
		if err != nil {
			return
		}
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

	if len(b.lockParts) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(b.lockParts, sql, " ", args)
		if err != nil {
			return
		}
	}

	sqlStr = sql.String()
	return
}

// ToCountBuilder rewrites the query into a COUNT query while preserving filters and joins.
func (b *SelectBuilder) ToCountBuilder() *SelectBuilder {
	hasDistinct := hasOption(b.options, "DISTINCT")

	if len(b.groupBys) == 0 && !hasDistinct {
		// A simple SELECT can count rows directly against the same FROM/WHERE clauses.
		builder := &SelectBuilder{
			withParts:  cloneSqlizers(b.withParts),
			prefixes:   cloneSqlizers(b.prefixes),
			columns:    nil,
			from:       b.from,
			joins:      cloneSqlizers(b.joins),
			whereParts: cloneSqlizers(b.whereParts),
			suffixes:   cloneSqlizers(b.suffixes),
		}
		return builder.Select("COUNT(1)")
	}

	if len(b.groupBys) == 1 &&
		len(b.havingParts) == 0 &&
		!strings.Contains(b.groupBys[0], ",") {
		// A single grouping key without HAVING can be reduced to COUNT(DISTINCT key).
		builder := &SelectBuilder{
			withParts:  cloneSqlizers(b.withParts),
			prefixes:   cloneSqlizers(b.prefixes),
			columns:    nil,
			from:       b.from,
			joins:      cloneSqlizers(b.joins),
			whereParts: cloneSqlizers(b.whereParts),
			suffixes:   cloneSqlizers(b.suffixes),
		}
		return builder.Select(fmt.Sprintf("COUNT(DISTINCT %s)", b.groupBys[0]))
	}

	// Complex GROUP BY or HAVING queries must be counted from a subquery to preserve semantics.
	subBuilder := &SelectBuilder{
		options:     cloneStrings(b.options),
		columns:     cloneSqlizers(b.columns),
		from:        b.from,
		joins:       cloneSqlizers(b.joins),
		whereParts:  cloneSqlizers(b.whereParts),
		groupBys:    cloneStrings(b.groupBys),
		havingParts: cloneSqlizers(b.havingParts),
	}

	builder := new(SelectBuilder).
		Select("COUNT(1)").
		FromSelect(subBuilder, "sub")
	builder.withParts = cloneSqlizers(b.withParts)
	builder.prefixes = cloneSqlizers(b.prefixes)
	builder.suffixes = cloneSqlizers(b.suffixes)
	return builder
}

// Prefix adds an expression to the beginning of the query
func (b *SelectBuilder) Prefix(sql string, args ...any) *SelectBuilder {
	if isWithClause(sql) {
		return b.With(sql, args...)
	}
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b *SelectBuilder) PrefixExpr(expr Sqlizer) *SelectBuilder {
	b.prefixes = append(b.prefixes, expr)
	return b
}

// With adds a WITH clause to the beginning of the query.
func (b *SelectBuilder) With(sql string, args ...any) *SelectBuilder {
	return b.WithExpr(Expr(sql, args...))
}

// WithExpr adds a WITH clause to the beginning of the query.
func (b *SelectBuilder) WithExpr(expr Sqlizer) *SelectBuilder {
	b.withParts = append(b.withParts, expr)
	return b
}

// Distinct adds a DISTINCT clause to the query.
func (b *SelectBuilder) Distinct() *SelectBuilder {
	return b.Options("DISTINCT")
}

// Options adds select option to the query
func (b *SelectBuilder) Options(options ...string) *SelectBuilder {
	b.options = append(b.options, options...)
	return b
}

// Select adds result columns to the query.
func (b *SelectBuilder) Select(columns ...string) *SelectBuilder {
	parts := make([]Sqlizer, 0, len(columns))
	for _, str := range columns {
		parts = append(parts, newPart(str))
	}
	b.columns = parts
	return b
}

// RemoveColumns remove all columns from query.
// Must add a new column with AddColumn or Select methods, otherwise
// return a error.
func (b *SelectBuilder) RemoveColumns() *SelectBuilder {
	b.columns = nil
	return b
}

// AddColumn adds a result column to the query.
// Unlike Select, AddColumn accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	AddColumn("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
func (b *SelectBuilder) AddColumn(column any, args ...any) *SelectBuilder {
	b.columns = append(b.columns, newPart(column, args...))
	return b
}

// From sets the FROM clause of the query.
func (b *SelectBuilder) From(from string) *SelectBuilder {
	b.from = newPart(from)
	return b
}

// FromSelect sets a subquery into the FROM clause of the query.
func (b *SelectBuilder) FromSelect(from *SelectBuilder, alias string) *SelectBuilder {
	b.from = Alias(from, alias)
	return b
}

// JoinClause adds a join clause to the query.
func (b *SelectBuilder) JoinClause(pred any, args ...any) *SelectBuilder {
	b.joins = append(b.joins, newPart(pred, args...))
	return b
}

// Join adds a JOIN clause to the query.
func (b *SelectBuilder) Join(join string, rest ...any) *SelectBuilder {
	return b.JoinClause("JOIN "+join, rest...)
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (b *SelectBuilder) LeftJoin(join string, rest ...any) *SelectBuilder {
	return b.JoinClause("LEFT JOIN "+join, rest...)
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (b *SelectBuilder) RightJoin(join string, rest ...any) *SelectBuilder {
	return b.JoinClause("RIGHT JOIN "+join, rest...)
}

// InnerJoin adds a INNER JOIN clause to the query.
func (b *SelectBuilder) InnerJoin(join string, rest ...any) *SelectBuilder {
	return b.JoinClause("INNER JOIN "+join, rest...)
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (b *SelectBuilder) CrossJoin(join string, rest ...any) *SelectBuilder {
	return b.JoinClause("CROSS JOIN "+join, rest...)
}

// Where adds an expression to the WHERE clause of the query.
//
// Expressions are ANDed together in the generated SQL.
//
// Where accepts several types for its pred argument:
//
// nil OR "" - ignored.
//
// string - SQL expression.
// If the expression has SQL placeholders then a set of arguments must be passed
// as well, one for each placeholderFormat.
//
// map[string]any OR Eq - map of SQL expressions to values. Each key is
// transformed into an expression like "<key> = ?", with the corresponding value
// bound to the placeholderFormat. If the value is nil, the expression will be "<key>
// IS NULL". Slices and arrays are not expanded automatically; use In or NotIn
// explicitly when you need an IN-style predicate.
//
// Where will panic if pred isn't any of the above types.
func (b *SelectBuilder) Where(pred any, args ...any) *SelectBuilder {
	if pred == nil || pred == "" {
		return b
	}
	b.whereParts = append(b.whereParts, newWherePart(pred, args...))
	return b
}

// GroupBy adds GROUP BY expressions to the query.
func (b *SelectBuilder) GroupBy(groupBys ...string) *SelectBuilder {
	b.groupBys = append(b.groupBys, groupBys...)
	return b
}

// Having adds an expression to the HAVING clause of the query.
//
// See Where.
func (b *SelectBuilder) Having(pred any, rest ...any) *SelectBuilder {
	b.havingParts = append(b.havingParts, newWherePart(pred, rest...))
	return b
}

// OrderByClause adds ORDER BY clause to the query.
func (b *SelectBuilder) OrderByClause(pred any, args ...any) *SelectBuilder {
	b.orderByParts = append(b.orderByParts, newPart(pred, args...))
	return b
}

// OrderBy adds ORDER BY expressions to the query.
func (b *SelectBuilder) OrderBy(orderBys ...string) *SelectBuilder {
	for _, orderBy := range orderBys {
		b.OrderByClause(orderBy)
	}

	return b
}

// Limit sets a LIMIT clause on the query.
func (b *SelectBuilder) Limit(limit uint64) *SelectBuilder {
	b.limit = strconv.FormatUint(limit, 10)
	return b
}

// RemoveLimit remove LIMIT clause
func (b *SelectBuilder) RemoveLimit() *SelectBuilder {
	b.limit = ""
	return b
}

// Offset sets a OFFSET clause on the query.
func (b *SelectBuilder) Offset(offset uint64) *SelectBuilder {
	b.offset = strconv.FormatUint(offset, 10)
	return b
}

// RemoveOffset removes OFFSET clause.
func (b *SelectBuilder) RemoveOffset() *SelectBuilder {
	b.offset = ""
	return b
}

// Suffix adds an expression to the end of the query
func (b *SelectBuilder) Suffix(sql string, args ...any) *SelectBuilder {
	if isLockClause(sql) {
		return b.Lock(sql, args...)
	}
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b *SelectBuilder) SuffixExpr(expr Sqlizer) *SelectBuilder {
	b.suffixes = append(b.suffixes, expr)
	return b
}

// Lock adds a row-locking clause such as FOR UPDATE to the end of the query.
func (b *SelectBuilder) Lock(sql string, args ...any) *SelectBuilder {
	return b.LockExpr(Expr(sql, args...))
}

// LockExpr adds a row-locking clause to the end of the query.
func (b *SelectBuilder) LockExpr(expr Sqlizer) *SelectBuilder {
	b.lockParts = append(b.lockParts, expr)
	return b
}

func isWithClause(sql string) bool {
	s := strings.TrimSpace(strings.ToUpper(sql))
	return s == "WITH" || strings.HasPrefix(s, "WITH ")
}

func isLockClause(sql string) bool {
	s := strings.TrimSpace(strings.ToUpper(sql))
	return s == "FOR UPDATE" ||
		strings.HasPrefix(s, "FOR UPDATE ") ||
		s == "FOR SHARE" ||
		strings.HasPrefix(s, "FOR SHARE ") ||
		s == "FOR KEY SHARE" ||
		strings.HasPrefix(s, "FOR KEY SHARE ") ||
		s == "FOR NO KEY UPDATE" ||
		strings.HasPrefix(s, "FOR NO KEY UPDATE ")
}

// Clear resets all fields while preserving slice capacity for reuse.
func (b *SelectBuilder) Clear() *SelectBuilder {
	b.withParts = resetSlice(b.withParts)
	b.prefixes = resetSlice(b.prefixes)
	b.options = resetSlice(b.options)
	b.columns = resetSlice(b.columns)
	b.from = nil
	b.joins = resetSlice(b.joins)
	b.whereParts = resetSlice(b.whereParts)
	b.groupBys = resetSlice(b.groupBys)
	b.havingParts = resetSlice(b.havingParts)
	b.orderByParts = resetSlice(b.orderByParts)
	b.limit = ""
	b.offset = ""
	b.suffixes = resetSlice(b.suffixes)
	b.lockParts = resetSlice(b.lockParts)
	return b
}
