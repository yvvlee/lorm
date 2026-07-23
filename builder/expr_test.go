package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcatExpr(t *testing.T) {
	b := ConcatExpr("COALESCE(name,", Expr("CONCAT(?,' ',?)", "f", "l"), ")")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "COALESCE(name,CONCAT(?,' ',?))"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"f", "l"}
	assert.Equal(t, expectedArgs, args)
}

func TestConcatExprBadType(t *testing.T) {
	b := ConcatExpr("prefix", 123, "suffix")
	_, _, err := b.ToSql()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "123 is not")
}

func TestEqToSql(t *testing.T) {
	b := Eq{"id": 1}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id = ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestEqEmptyToSql(t *testing.T) {
	sql, args, err := Eq{}.ToSql()
	assert.NoError(t, err)

	expectedSql := "(1=1)"
	assert.Equal(t, expectedSql, sql)
	assert.Empty(t, args)
}

func TestEqNilToSql(t *testing.T) {
	sql, args, err := Eq{"name": nil}.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "name = ?", sql)
	assert.Equal(t, []any{nil}, args)
}

func TestEqSliceToSql(t *testing.T) {
	b := Eq{"id": []int{1, 2, 3}}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id = ?", sql)
	assert.Equal(t, []any{[]int{1, 2, 3}}, args)
}

func TestNotEqSliceToSql(t *testing.T) {
	b := NotEq{"id": []int{1, 2, 3}}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id <> ?", sql)
	assert.Equal(t, []any{[]int{1, 2, 3}}, args)
}

func TestEqEmptySliceToSql(t *testing.T) {
	b := Eq{"id": []int{}}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id = ?", sql)
	assert.Equal(t, []any{[]int{}}, args)
}

func TestNotEqEmptySliceToSql(t *testing.T) {
	b := NotEq{"id": []int{}}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id <> ?", sql)
	assert.Equal(t, []any{[]int{}}, args)
}

func TestNotEqToSql(t *testing.T) {
	b := NotEq{"id": 1}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id <> ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestInToSql(t *testing.T) {
	b := In("id", []int{1, 2, 3})
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id IN (?,?,?)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1, 2, 3}
	assert.Equal(t, expectedArgs, args)
}

func TestNotInToSql(t *testing.T) {
	b := NotIn("id", []int{1, 2, 3})
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id NOT IN (?,?,?)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1, 2, 3}
	assert.Equal(t, expectedArgs, args)
}

func TestInEmptyToSql(t *testing.T) {
	b := In("id", []int{})
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "(1=0)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{}
	assert.Equal(t, expectedArgs, args)
}

func TestNotInEmptyToSql(t *testing.T) {
	b := NotIn("id", []int{})
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "(1=1)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{}
	assert.Equal(t, expectedArgs, args)
}

func TestEqBytesToSql(t *testing.T) {
	b := Eq{"id": []byte("test")}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id = ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{[]byte("test")}
	assert.Equal(t, expectedArgs, args)
}

func TestLtToSql(t *testing.T) {
	b := Lt("id", 1)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id < ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestLtOrEqToSql(t *testing.T) {
	b := Lte("id", 1)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id <= ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestGtToSql(t *testing.T) {
	b := Gt("id", 1)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id > ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestGtOrEqToSql(t *testing.T) {
	b := Gte("id", 1)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "id >= ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1}
	assert.Equal(t, expectedArgs, args)
}

func TestIsNullToSql(t *testing.T) {
	sql, args, err := IsNull("name").ToSql()
	assert.NoError(t, err)
	assert.Empty(t, args)
	assert.Equal(t, "name IS NULL", sql)
}

func TestIsNotNullToSql(t *testing.T) {
	sql, args, err := IsNotNull("name").ToSql()
	assert.NoError(t, err)
	assert.Empty(t, args)
	assert.Equal(t, "name IS NOT NULL", sql)
}

func TestEmptyAndToSql(t *testing.T) {
	sql, args, err := And{}.ToSql()
	assert.NoError(t, err)

	expectedSql := "(1=1)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{}
	assert.Equal(t, expectedArgs, args)
}

func TestEmptyOrToSql(t *testing.T) {
	sql, args, err := Or{}.ToSql()
	assert.NoError(t, err)

	expectedSql := "(1=0)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{}
	assert.Equal(t, expectedArgs, args)
}

func TestConjunctionIgnoresNilPredicates(t *testing.T) {
	tests := []struct {
		name string
		pred Sqlizer
		want string
	}{
		{name: "empty and", pred: And{nil}, want: sqlTrue},
		{name: "empty or", pred: Or{nil}, want: sqlFalse},
		{name: "mixed and", pred: And{nil, Eq{"id": 1}}, want: "(id = ?)"},
		{name: "mixed or", pred: Or{nil, Eq{"id": 1}}, want: "(id = ?)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := tt.pred.ToSql()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, sql)
		})
	}
}

func TestLikeToSql(t *testing.T) {
	b := Like("name", "%irrel")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "name LIKE ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"%irrel"}
	assert.Equal(t, expectedArgs, args)
}

func TestNotLikeToSql(t *testing.T) {
	b := NotLike("name", "%irrel")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "name NOT LIKE ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"%irrel"}
	assert.Equal(t, expectedArgs, args)
}

func TestILikeToSql(t *testing.T) {
	b := ILike("name", "sq%")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "name ILIKE ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"sq%"}
	assert.Equal(t, expectedArgs, args)
}

func TestNotILikeToSql(t *testing.T) {
	b := NotILike("name", "sq%")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "name NOT ILIKE ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"sq%"}
	assert.Equal(t, expectedArgs, args)
}

func TestSqlEqOrder(t *testing.T) {
	b := Eq{"a": 1, "b": 2, "c": 3}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "a = ? AND b = ? AND c = ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1, 2, 3}
	assert.Equal(t, expectedArgs, args)
}

func TestSqlLtOrder(t *testing.T) {
	b := And{Lt("a", 1), Lt("b", 2), Lt("c", 3)}
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "(a < ? AND b < ? AND c < ?)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{1, 2, 3}
	assert.Equal(t, expectedArgs, args)
}

func TestExprEscaped(t *testing.T) {
	b := Expr("count(??)", Expr("x"))
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "count(??)"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{Expr("x")}
	assert.Equal(t, expectedArgs, args)
}

func TestExprRecursion(t *testing.T) {
	{
		b := Expr("count(?)", Expr("nullif(a,?)", "b"))
		sql, args, err := b.ToSql()
		assert.NoError(t, err)

		expectedSql := "count(nullif(a,?))"
		assert.Equal(t, expectedSql, sql)

		expectedArgs := []any{"b"}
		assert.Equal(t, expectedArgs, args)
	}
	{
		b := Expr("extract(? from ?)", Expr("epoch"), "2001-02-03")
		sql, args, err := b.ToSql()
		assert.NoError(t, err)

		expectedSql := "extract(epoch from ?)"
		assert.Equal(t, expectedSql, sql)

		expectedArgs := []any{"2001-02-03"}
		assert.Equal(t, expectedArgs, args)
	}
	{
		b := Expr("JOIN t1 ON ?", And{Eq{"id": 1}, Expr("NOT c1"), Expr("? @@ ?", "x", "y")})
		sql, args, err := b.ToSql()
		assert.NoError(t, err)

		expectedSql := "JOIN t1 ON (id = ? AND NOT c1 AND ? @@ ?)"
		assert.Equal(t, expectedSql, sql)

		expectedArgs := []any{1, "x", "y"}
		assert.Equal(t, expectedArgs, args)
	}
}

func TestBetweenIntToSql(t *testing.T) {
	b := Between("age", 18, 65)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "age BETWEEN ? AND ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{18, 65}
	assert.Equal(t, expectedArgs, args)
}

func TestBetweenFloat64ToSql(t *testing.T) {
	b := Between("price", 10.5, 99.99)
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "price BETWEEN ? AND ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{10.5, 99.99}
	assert.Equal(t, expectedArgs, args)
}

func TestBetweenStringToSql(t *testing.T) {
	b := Between("name", "A", "Z")
	sql, args, err := b.ToSql()
	assert.NoError(t, err)

	expectedSql := "name BETWEEN ? AND ?"
	assert.Equal(t, expectedSql, sql)

	expectedArgs := []any{"A", "Z"}
	assert.Equal(t, expectedArgs, args)
}

func ExampleEq() {
	Select("id", "created", "first_name").From("users").Where(Eq{
		"company": 20,
	})
}
