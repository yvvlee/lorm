package builder

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionGroupsPreserveEmptyTrueAndFalse(t *testing.T) {
	for _, tt := range []struct {
		name string
		pred Sqlizer
		sql  string
		args []any
	}{
		{"empty or", Or{}, "", nil},
		{"all absent", Or{nil, Expr(""), Or{nil, Or{}}}, "", nil},
		{"absent in and", And{Eq{"id": 7}, Or{}}, "(id = ?)", []any{7}},
		{"absent in or", Or{Or{}, Eq{"id": 7}}, "(id = ?)", []any{7}},
		{"empty and remains true", And{Or{}}, sqlTrue, nil},
		{"true in and", And{Eq{}, Eq{"id": 7}}, "(id = ?)", []any{7}},
		{"true in or", Or{Eq{"id": 7}, Eq{}}, sqlTrue, nil},
		{"false in and", And{Eq{"id": 7}, In("id", []int{})}, sqlFalse, nil},
		{"false in or", Or{In("id", []int{}), Eq{"id": 7}}, "(id = ?)", []any{7}},
		{"only false", Or{Or{}, In("id", []int{}), In("name", []string{})}, sqlFalse, nil},
		{"only true", And{Eq{}, NotEq{}}, sqlTrue, nil},
		{"empty not in remains true", Or{NotIn("id", []int{}), Eq{"id": 7}}, sqlTrue, nil},
		{"nested true in or", Or{And{Eq{}, Eq{}}, Eq{"id": 7}}, sqlTrue, nil},
		{"nested true in and", And{Or{Eq{}, Eq{"id": 7}}, Eq{"tenant": 2}}, "(tenant = ?)", []any{2}},
		{"nested false in or", Or{And{In("id", []int{}), Eq{"id": 7}}, Eq{"tenant": 2}}, "(tenant = ?)", []any{2}},
		{"true or false", Or{Eq{}, In("id", []int{})}, sqlTrue, nil},
		{"true and false", And{Eq{}, In("id", []int{})}, sqlFalse, nil},
		{"standalone literal", And{Expr(" (( 1 = 1 )) "), Expr("TRUE"), Eq{"id": 7}}, "(id = ?)", []any{7}},
		{"raw expression stays opaque", Or{Expr("1=1 OR status = ?", 3), Eq{"id": 7}}, "(1=1 OR status = ? OR id = ?)", []any{3, 7}},
		{"argument order", And{Eq{"tenant": 2}, Or{In("id", []int{}), Eq{"id": 7}, Eq{"status": 3}}, Or{}}, "(tenant = ? AND (id = ? OR status = ?))", []any{2, 7, 3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.pred.ToSql()
			require.NoError(t, err)
			assert.Equal(t, tt.sql, sql)
			assert.Equal(t, tt.args, append([]any(nil), args...))
		})
	}
}

func TestConditionClausesOmitOnlyEmptyAndTrue(t *testing.T) {
	for _, tt := range []struct {
		name  string
		pred  any
		where string
		args  []any
	}{
		{"empty or", Or{}, "", nil},
		{"nested empty or", Or{nil, Or{Expr("")}}, "", nil},
		{"empty expression", Expr(""), "", nil},
		{"empty map", map[string]any{}, "", nil},
		{"true and", And{Eq{}, Eq{}}, "", nil},
		{"true or", Or{Eq{}, Eq{"id": 7}}, "", nil},
		{"false", Or{In("id", []int{})}, " WHERE (1=0)", nil},
		{"filtered", Or{Or{}, Eq{"id": 7}}, " WHERE (id = ?)", []any{7}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			selectSQL, args, err := Select("id").From("users").Where(tt.pred).OrderBy("id").ToSql()
			require.NoError(t, err)
			assert.Equal(t, "SELECT id FROM users"+tt.where+" ORDER BY id", selectSQL)
			assert.Equal(t, tt.args, append([]any(nil), args...))

			update := Update("users").Set("status", 3).Where(tt.pred)
			updateSQL, args, err := update.ToSql()
			require.NoError(t, err)
			assert.Equal(t, "UPDATE users SET status = ?"+tt.where, updateSQL)
			assert.Equal(t, append([]any{3}, tt.args...), args)
			assert.Equal(t, tt.where != "", update.HasWhere())

			delete := Delete("users").Where(tt.pred)
			deleteSQL, args, err := delete.ToSql()
			require.NoError(t, err)
			assert.Equal(t, "DELETE FROM users"+tt.where, deleteSQL)
			assert.Equal(t, tt.args, append([]any(nil), args...))
			assert.Equal(t, tt.where != "", delete.HasWhere())

			// Multiple Where calls are an implicit AND; a true group cannot remove a sibling restriction.
			sql, args, err := Delete("users").Where(tt.pred).Where(Eq{"tenant": 9}).ToSql()
			require.NoError(t, err)
			if tt.name == "false" {
				assert.Equal(t, "DELETE FROM users WHERE (1=0)", sql)
				assert.Empty(t, args)
			} else if tt.where == "" {
				assert.Equal(t, "DELETE FROM users WHERE tenant = ?", sql)
				assert.Equal(t, []any{9}, args)
			} else {
				assert.Equal(t, "DELETE FROM users"+tt.where+" AND tenant = ?", sql)
				assert.Equal(t, append(append([]any(nil), tt.args...), 9), args)
			}
		})
	}
}

func TestHavingEmptyOrAndCountBuilder(t *testing.T) {
	stmt := Select("status").From("users").GroupBy("status").Having(Or{nil, Or{}})
	query, args, err := stmt.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT status FROM users GROUP BY status", query)
	assert.Empty(t, args)
	query, args, err = stmt.ToCountBuilder().ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT COUNT(1) FROM (SELECT status FROM users GROUP BY status) AS sub", query)
	assert.Empty(t, args)
	query, args, err = stmt.Having(Or{In("id", []int{})}).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT status FROM users GROUP BY status HAVING (1=0)", query)
	assert.Empty(t, args)
}

func TestHavingRetainsTrueForGroupingSemantics(t *testing.T) {
	stmt := Select("1").From("users").Having(Or{Eq{}, Eq{"id": 7}})
	query, args, err := stmt.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1 FROM users HAVING (1=1)", query)
	assert.Empty(t, args)
	query, args, err = stmt.ToCountBuilder().ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT COUNT(1) FROM (SELECT 1 FROM users HAVING (1=1)) AS sub", query)
	assert.Empty(t, args)
}

type failingCondition struct{ err error }

func (c failingCondition) ToSql() (string, []any, error) { return "", nil, c.err }

func TestConstantsDoNotHideConditionErrors(t *testing.T) {
	want := errors.New("invalid condition")
	for _, pred := range []Sqlizer{
		Or{Eq{}, failingCondition{want}},
		And{In("id", []int{}), failingCondition{want}},
		Or{Or{}, And{Eq{}, failingCondition{want}}},
	} {
		_, _, err := pred.ToSql()
		require.ErrorIs(t, err, want)
		_, _, err = Select("id").From("users").Where(pred).ToSql()
		require.ErrorIs(t, err, want)
		_, _, err = Update("users").Set("status", 1).Where(pred).ToSql()
		require.ErrorIs(t, err, want)
		_, _, err = Delete("users").Where(pred).ToSql()
		require.ErrorIs(t, err, want)
	}
}
