package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteRejectsEmptyValueExpressions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value Sqlizer
	}{
		{"empty", Expr("")},
		{"whitespace", Expr(" \t\n")},
		{"empty_or", Or{}},
		{"nested_empty_or", Or{Or{}}},
		{"empty_with_args", Expr("", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Insert("users").Columns("id", "name").Values(1, "ok").Values(2, tc.value).ToSql()
			require.ErrorContains(t, err, "row 1, column 1 must not be empty")
			_, _, err = Update("users").Set("name", tc.value).Where(Eq{"id": 1}).ToSql()
			require.ErrorContains(t, err, `column "name" must not be empty`)
		})
	}
}

func TestWriteAllowsNullAndBooleanValueExpressions(t *testing.T) {
	query, args, err := Insert("users").Columns("name", "enabled", "note").Values(Expr("NULL"), And{}, nil).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users (name,enabled,note) VALUES (NULL,(1=1),?)", query)
	assert.Equal(t, []any{nil}, args)
	query, args, err = Update("users").Set("name", Expr("NULL")).Set("enabled", And{}).Set("note", nil).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "UPDATE users SET name = NULL, enabled = (1=1), note = ?", query)
	assert.Equal(t, []any{nil}, args)
}

func TestInsertValuesWrapsSubqueries(t *testing.T) {
	query, args, err := Insert("users").Columns("id", "name").
		Values(1, Select("name").From("source").Where(Eq{"id": 2})).
		Values(Expr("? + 1", 3), Expr("UPPER(?)", "name")).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users (id,name) VALUES (?,(SELECT name FROM source WHERE id = ?)),(? + 1,UPPER(?))", query)
	assert.Equal(t, []any{1, 2, 3, "name"}, args)
	_, _, err = Insert("users").Columns("name").Values(Select()).ToSql()
	require.ErrorContains(t, err, "at least one result column")
}
