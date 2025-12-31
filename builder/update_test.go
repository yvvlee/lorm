package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateBuilderToSql(t *testing.T) {
	b := Update("").
		Prefix("WITH prefix AS ?", 0).
		Table("a").
		Set("b", Expr("? + 1", 1)).
		SetMap(Eq{"c": 2}).
		Set("c1", Case("status").When("1", "2").When("2", "1")).
		Set("c2", Case().When("a = 2", Expr("?", "foo")).When("a = 3", Expr("?", "bar"))).
		Set("c3", Select("a").From("b")).
		Where("d = ?", 3).
		OrderBy("e").
		Limit(4).
		Offset(5).
		Suffix("RETURNING ?", 6)

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql :=
		"WITH prefix AS ? " +
			"UPDATE a SET b = ? + 1, c = ?, " +
			"c1 = CASE status WHEN 1 THEN 2 WHEN 2 THEN 1 END, " +
			"c2 = CASE WHEN a = 2 THEN ? WHEN a = 3 THEN ? END, " +
			"c3 = (SELECT a FROM b) " +
			"WHERE d = ? " +
			"ORDER BY e LIMIT 4 OFFSET 5 " +
			"RETURNING ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{0, 1, 2, "foo", "bar", 3, 6}
	assert.Equal(t, expectedArgs, args)
}

func TestUpdateBuilderToSqlErr(t *testing.T) {
	_, _, err := Update("").Set("x", 1).ToSql()
	assert.Error(t, err)

	_, _, err = Update("x").ToSql()
	assert.Error(t, err)
}

func TestUpdateBuilderPlaceholders(t *testing.T) {
	b := Update("test").SetMap(Eq{"x": 1, "y": 2})

	sql, _, _ := b.ToSql()
	assert.Equal(t, "UPDATE test SET x = ?, y = ?", sql)

	sql, _, _ = b.ToSql()
	sql, _ = Dollar.ReplacePlaceholders(sql)
	assert.Equal(t, "UPDATE test SET x = $1, y = $2", sql)
}

func TestUpdateBuilderFrom(t *testing.T) {
	sql, _, err := Update("employees").Set("sales_count", 100).From("accounts").Where("accounts.name = ?", "ACME").ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "UPDATE employees SET sales_count = ? FROM accounts WHERE accounts.name = ?", sql)
}

func TestUpdateBuilderFromSelect(t *testing.T) {
	sql, _, err := Update("employees").
		Set("sales_count", 100).
		FromSelect(Select("id").
			From("accounts").
			Where("accounts.name = ?", "ACME"), "subquery").
		Where("employees.account_id = subquery.id").ToSql()
	assert.NoError(t, err)

	expectedSql :=
		"UPDATE employees " +
			"SET sales_count = ? " +
			"FROM (SELECT id FROM accounts WHERE accounts.name = ?) AS subquery " +
			"WHERE employees.account_id = subquery.id"
	assert.Equal(t, expectedSql, sql)
}

func TestUpdateBuilderReturningWithSuffix(t *testing.T) {
	b := Update("").
		Table("users").
		Set("name", "John").
		Set("email", "john@example.com").
		Where("id = ?", 1).
		Suffix("FROM other_table").
		Returning("id", "updated_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "UPDATE users SET name = ?, email = ? WHERE id = ? FROM other_table RETURNING id,updated_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"John", "john@example.com", 1}
	assert.Equal(t, expectedArgs, args)
}

func TestUpdateBuilderReturningWithoutSuffix(t *testing.T) {
	b := Update("").
		Table("users").
		Set("status", "active").
		Where("id = ?", 1).
		Returning("id", "status", "updated_at")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "UPDATE users SET status = ? WHERE id = ? RETURNING id,status,updated_at"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{"active", 1}
	assert.Equal(t, expectedArgs, args)
}

func TestUpdateBuilderReturningWithMultipleSuffixes(t *testing.T) {
	b := Update("").
		Table("products").
		Set("price", 99.99).
		Where("category = ?", "electronics").
		Suffix("LIMIT 10").
		Suffix("OFFSET 5").
		Returning("id", "name", "price")

	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSQL := "UPDATE products SET price = ? WHERE category = ? LIMIT 10 OFFSET 5 RETURNING id,name,price"
	assert.Equal(t, expectedSQL, sql)

	expectedArgs := []any{99.99, "electronics"}
	assert.Equal(t, expectedArgs, args)
}
