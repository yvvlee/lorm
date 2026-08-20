package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteBuilderToSql(t *testing.T) {
	b := Delete("").
		Prefix("WITH prefix AS ?", 0).
		From("a").
		Where("b = ?", 1).
		OrderBy("c").
		Limit(2).
		Offset(3).
		Suffix("RETURNING ?", 4)

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql :=
		"WITH prefix AS ? " +
			"DELETE FROM a WHERE b = ? ORDER BY c LIMIT 2 OFFSET 3 " +
			"RETURNING ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{0, 1, 4}
	assert.Equal(t, expectedArgs, args)
}

func TestDeleteBuilderToSqlErr(t *testing.T) {
	_, _, err := Delete("").ToSql()
	assert.Error(t, err)
}

func TestDeleteBuilderIgnoresNilAndEmptyWhereClauses(t *testing.T) {
	sql, args, err := Delete("users").
		Where(nil).
		Where("").
		ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "DELETE FROM users", sql)
	assert.Empty(t, args)
}

func TestDeleteBuilderHasEffectiveWhere(t *testing.T) {
	assert.False(t, Delete("users").Where(nil).HasWhere())
	assert.False(t, Delete("users").Where(Eq{}).HasWhere())
	assert.False(t, Delete("users").Where("(1 = 1)").HasWhere())
	assert.True(t, Delete("users").Where("id = ?", 1).HasWhere())
}

func TestDeleteBuilderReturningWithSuffix(t *testing.T) {
	b := Delete("").
		From("users").
		Where("status = ?", "inactive").
		Suffix("USING other_table").
		Returning("id", "deleted_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "DELETE FROM users WHERE status = ? USING other_table RETURNING id,deleted_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"inactive"}
	assert.Equal(t, expectedArgs, args)
}

func TestDeleteBuilderReturningWithoutSuffix(t *testing.T) {
	b := Delete("").
		From("logs").
		Where("created_at < ?", "2024-01-01").
		Returning("id", "created_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "DELETE FROM logs WHERE created_at < ? RETURNING id,created_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"2024-01-01"}
	assert.Equal(t, expectedArgs, args)
}

func TestDeleteBuilderReturningWithMultipleSuffixes(t *testing.T) {
	b := Delete("").
		From("sessions").
		Where("expired = ?", true).
		Suffix("LIMIT 100").
		Suffix("OFFSET 10").
		Returning("id", "user_id", "expired_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "DELETE FROM sessions WHERE expired = ? LIMIT 100 OFFSET 10 RETURNING id,user_id,expired_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{true}
	assert.Equal(t, expectedArgs, args)
}

func TestDeleteBuilderCloneDoesNotMutateSourceBuilder(t *testing.T) {
	b := Delete("users").
		Where("status = ?", "inactive").
		Limit(10)

	clone := b.Clone().
		Where("tenant_id = ?", 2).
		Limit(1)

	cloneSQL, cloneArgs, err := clone.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "DELETE FROM users WHERE status = ? AND tenant_id = ? LIMIT 1", cloneSQL)
	assert.Equal(t, []any{"inactive", 2}, cloneArgs)

	sql, args, err := b.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "DELETE FROM users WHERE status = ? LIMIT 10", sql)
	assert.Equal(t, []any{"inactive"}, args)
}
