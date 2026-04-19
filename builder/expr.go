package builder

import (
	"bytes"
	"cmp"
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/samber/lo"
)

const (
	// Portable true/false literals.
	sqlTrue  = "(1=1)"
	sqlFalse = "(1=0)"
)

type expr struct {
	sql  string
	args []any
}

// Expr builds an expression from a SQL fragment and arguments.
//
// Ex:
//
//	Expr("FROM_UNIXTIME(?)", t)
func Expr(sql string, args ...any) Sqlizer {
	return expr{sql: sql, args: args}
}

func (e expr) ToSql() (sql string, args []any, err error) {
	simple := true
	for _, arg := range e.args {
		if _, ok := arg.(Sqlizer); ok {
			simple = false
		}
	}
	if simple {
		return e.sql, e.args, nil
	}

	buf := &bytes.Buffer{}
	ap := e.args
	sp := e.sql

	var isql string
	var iargs []any

	for err == nil && len(ap) > 0 && len(sp) > 0 {
		i := strings.Index(sp, "?")
		if i < 0 {
			// no more placeholders
			break
		}
		if len(sp) > i+1 && sp[i+1:i+2] == "?" {
			// escaped "??"; append it and step past
			buf.WriteString(sp[:i+2])
			sp = sp[i+2:]
			continue
		}

		if as, ok := ap[0].(Sqlizer); ok {
			// sqlizer argument; expand it and append the result
			isql, iargs, err = as.ToSql()
			buf.WriteString(sp[:i])
			buf.WriteString(isql)
			args = append(args, iargs...)
		} else {
			// normal argument; append it and the placeholderFormat
			buf.WriteString(sp[:i+1])
			args = append(args, ap[0])
		}

		// step past the argument and placeholderFormat
		ap = ap[1:]
		sp = sp[i+1:]
	}

	// append the remaining sql and arguments
	buf.WriteString(sp)
	return buf.String(), append(args, ap...), err
}

type concatExpr []any

func (ce concatExpr) ToSql() (sql string, args []any, err error) {
	for _, part := range ce {
		switch p := part.(type) {
		case string:
			sql += p
		case Sqlizer:
			pSql, pArgs, err := p.ToSql()
			if err != nil {
				return "", nil, err
			}
			sql += pSql
			args = append(args, pArgs...)
		default:
			return "", nil, fmt.Errorf("%#v is not a string or Sqlizer", part)
		}
	}
	return
}

// ConcatExpr builds an expression by concatenating strings and other expressions.
//
// Ex:
//
//	name_expr := Expr("CONCAT(?, ' ', ?)", firstName, lastName)
//	ConcatExpr("COALESCE(full_name,", name_expr, ")")
func ConcatExpr(parts ...any) Sqlizer {
	return concatExpr(parts)
}

// aliasExpr helps to alias part of SQL query generated with underlying "expr"
type aliasExpr struct {
	expr  Sqlizer
	alias string
}

// Alias allows to define alias for column in SelectBuilder. Useful when column is
// defined as complex expression like IF or CASE
// Ex:
//
//	.AddColumn(Alias(caseStmt, "case_column"))
func Alias(expr Sqlizer, alias string) Sqlizer {
	return aliasExpr{expr, alias}
}

func (e aliasExpr) ToSql() (sql string, args []any, err error) {
	sql, args, err = e.expr.ToSql()
	if err == nil {
		sql = fmt.Sprintf("(%s) AS %s", sql, e.alias)
	}
	return
}

// Eq is syntactic sugar for use with Where/Having/Set methods.
type Eq map[string]any

func (eq Eq) toSQL(useNotOpr bool) (sql string, args []any, err error) {
	if len(eq) == 0 {
		// Empty Sql{} evaluates to true.
		sql = sqlTrue
		return
	}

	var (
		exprs    []Sqlizer
		equalOpr = "="
		nullOpr  = "IS"
	)

	if useNotOpr {
		equalOpr = "<>"
		nullOpr = "IS NOT"
	}
	keys := lo.Keys(eq)
	slices.Sort(keys)
	for _, key := range keys {
		var e Sqlizer
		val := eq[key]

		switch v := val.(type) {
		case driver.Valuer:
			if val, err = v.Value(); err != nil {
				return
			}
		}

		r := reflect.ValueOf(val)
		if r.Kind() == reflect.Ptr {
			if r.IsNil() {
				val = nil
			} else {
				val = r.Elem().Interface()
				r = reflect.ValueOf(val)
			}
		}

		if val == nil {
			e = Expr(fmt.Sprintf("%s %s NULL", key, nullOpr))
		} else {
			if r.Kind() == reflect.Slice || r.Kind() == reflect.Array {
				if _, ok := val.([]byte); !ok {
					err = fmt.Errorf("cannot use array or slice with Eq operators")
					return
				}
			}
			e = Expr(fmt.Sprintf("%s %s ?", key, equalOpr), val)
		}
		exprs = append(exprs, e)
	}

	var sqlParts []string
	for _, sqlizer := range exprs {
		partSQL, partArgs, err := sqlizer.ToSql()
		if err != nil {
			return "", nil, err
		}
		if partSQL != "" {
			sqlParts = append(sqlParts, partSQL)
			args = append(args, partArgs...)
		}
	}
	if len(sqlParts) > 0 {
		sql = strings.Join(sqlParts, " AND ")
	}
	return
}

func (eq Eq) ToSql() (sql string, args []any, err error) {
	return eq.toSQL(false)
}

// NotEq is syntactic sugar for use with Where/Having/Set methods.
// Ex:
//
//	.Where(NotEq{"id": 1}) == "id <> 1"
type NotEq Eq

func (neq NotEq) ToSql() (sql string, args []any, err error) {
	return Eq(neq).toSQL(true)
}

// In builds a field IN (...) predicate.
//
// An empty slice becomes a portable false expression because SQL does not allow IN ().
func In[T any](field string, val []T) Sqlizer {
	if len(val) == 0 {
		return expr{sql: sqlFalse, args: []any{}}
	}
	s := lo.Map(val, func(item T, index int) any {
		return any(item)
	})
	return Expr(fmt.Sprintf("%s IN (%s)", field, Placeholders(len(val))), s...)
}

// NotIn builds a field NOT IN (...) predicate.
//
// An empty slice becomes a portable true expression because SQL does not allow NOT IN ().
func NotIn[T any](field string, val []T) Sqlizer {
	if len(val) == 0 {
		return expr{sql: sqlTrue, args: []any{}}
	}
	s := lo.Map(val, func(item T, index int) any {
		return any(item)
	})
	return Expr(fmt.Sprintf("%s NOT IN (%s)", field, Placeholders(len(val))), s...)
}

// Like is syntactic sugar for use with LIKE conditions.
// Ex:
//
//	.Where(Like("name", "%irrel"))
func Like(field, value string) Sqlizer {
	return Expr(fmt.Sprintf("%s LIKE ?", field), value)
}

// NotLike is syntactic sugar for use with LIKE conditions.
// Ex:
//
//	.Where(NotLike("name": "%irrel"))
func NotLike(field, value string) Sqlizer {
	return Expr(fmt.Sprintf("%s NOT LIKE ?", field), value)
}

// ILike is syntactic sugar for use with ILIKE conditions.
// Ex:
//
//	.Where(ILike("name", "sq%"))
func ILike(field, value string) Sqlizer {
	return Expr(fmt.Sprintf("%s ILIKE ?", field), value)
}

// NotILike is syntactic sugar for use with ILIKE conditions.
// Ex:
//
//	.Where(NotILike("name", "sq%"))
func NotILike(field, value string) Sqlizer {
	return Expr(fmt.Sprintf("%s NOT ILIKE ?", field), value)
}

// Lt is syntactic sugar for use with Where/Having/Set methods.
// Ex:
//
//	.Where(Lt("id", 1))
func Lt[T cmp.Ordered](field string, value T) Sqlizer {
	return Expr(fmt.Sprintf("%s < ?", field), value)
}

// Lte is syntactic sugar for use with Where/Having/Set methods.
// Ex:
//
//	.Where(Lte("id", 1)) == "id <= 1"
func Lte[T cmp.Ordered](field string, value T) Sqlizer {
	return Expr(fmt.Sprintf("%s <= ?", field), value)
}

// Gt is syntactic sugar for use with Where/Having/Set methods.
// Ex:
//
//	.Where(Gt("id", 1)) == "id > 1"
func Gt[T cmp.Ordered](field string, value T) Sqlizer {
	return Expr(fmt.Sprintf("%s > ?", field), value)
}

// Gte is syntactic sugar for use with Where/Having/Set methods.
// Ex:
//
//	.Where(Gte("id", 1)) == "id >= 1"
func Gte[T cmp.Ordered](field string, value T) Sqlizer {
	return Expr(fmt.Sprintf("%s >= ?", field), value)
}

// Between builds a field BETWEEN ? AND ? predicate.
func Between[T cmp.Ordered](field string, start, end T) Sqlizer {
	return Expr(fmt.Sprintf("%s BETWEEN ? AND ?", field), start, end)
}

type conj []Sqlizer

func (c conj) join(sep, defaultExpr string) (sql string, args []any, err error) {
	if len(c) == 0 {
		return defaultExpr, []any{}, nil
	}
	var sqlParts []string
	for _, sqlizer := range c {
		partSQL, partArgs, err := sqlizer.ToSql()
		if err != nil {
			return "", nil, err
		}
		if partSQL != "" {
			sqlParts = append(sqlParts, partSQL)
			args = append(args, partArgs...)
		}
	}
	if len(sqlParts) > 0 {
		sql = fmt.Sprintf("(%s)", strings.Join(sqlParts, sep))
	}
	return
}

// And joins predicates with AND and wraps the result in parentheses.
type And conj

func (a And) ToSql() (string, []any, error) {
	return conj(a).join(" AND ", sqlTrue)
}

// Or joins predicates with OR and wraps the result in parentheses.
type Or conj

func (o Or) ToSql() (string, []any, error) {
	return conj(o).join(" OR ", sqlFalse)
}
