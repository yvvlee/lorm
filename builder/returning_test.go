package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertBuilderReturning(t *testing.T) {
	b := Insert("users").
		Columns("name", "email").
		Values("John", "john@example.com").
		Returning("id")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "INSERT INTO users (name,email) VALUES (?,?) RETURNING id"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"John", "john@example.com"}
	assert.Equal(t, expectedArgs, args)
}

func TestInsertBuilderReturningMultiple(t *testing.T) {
	b := Insert("users").
		Columns("name", "email").
		Values("John", "john@example.com").
		Returning("id", "created_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "INSERT INTO users (name,email) VALUES (?,?) RETURNING id,created_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"John", "john@example.com"}
	assert.Equal(t, expectedArgs, args)
}

func TestUpdateBuilderReturning(t *testing.T) {
	b := Update("users").
		Set("name", "Jane").
		Where(Eq{"id": 1}).
		Returning("updated_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "UPDATE users SET name = ? WHERE id = ? RETURNING updated_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"Jane", 1}
	assert.Equal(t, expectedArgs, args)
}

func TestDeleteBuilderReturning(t *testing.T) {
	b := Delete("users").
		Where(Eq{"id": 1}).
		Returning("id", "name")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "DELETE FROM users WHERE id = ? RETURNING id,name"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}
