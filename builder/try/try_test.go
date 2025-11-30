package try

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	val := 10
	c := Equal("age", &val)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age = ?", sql)
	assert.Equal(t, []any{10}, args)

	c = Equal("age", (*int)(nil))
	assert.Nil(t, c)
}

func TestLike(t *testing.T) {
	c := Like("name", "foo")
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "name LIKE ?", sql)
	assert.Equal(t, []any{"foo"}, args)

	c = Like("name", "  ")
	assert.Nil(t, c)
}

func TestIn(t *testing.T) {
	vals := []int{1, 2, 3}
	c := In("id", &vals)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id IN (?,?,?)", sql)
	assert.Equal(t, []any{1, 2, 3}, args)

	c = In("id", (*[]int)(nil))
	assert.Nil(t, c)

	emptyVals := []int{}
	c = In("id", &emptyVals)
	assert.Nil(t, c)
}

func TestRange(t *testing.T) {
	min := 10
	max := 20
	c := Range("age", &min, &max)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "(age >= ? AND age <= ?)", sql)
	assert.Equal(t, []any{10, 20}, args)

	c = Range("age", &min, nil)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age >= ?", sql)
	assert.Equal(t, []any{10}, args)

	c = Range("age", nil, &max)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age <= ?", sql)
	assert.Equal(t, []any{20}, args)

	c = Range("age", (*int)(nil), (*int)(nil))
	assert.Nil(t, c)
}

func TestTimeRange(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	c := TimeRange("created_at", &start, &end)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "(created_at >= ? AND created_at < ?)", sql)
	assert.Equal(t, []any{"2023-01-01 00:00:00", "2023-01-02 00:00:00"}, args)
}

func TestNotEqual(t *testing.T) {
	val := 10
	c := NotEqual("age", &val)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age <> ?", sql)
	assert.Equal(t, []any{10}, args)

	c = NotEqual("age", (*int)(nil))
	assert.Nil(t, c)
}

func TestGtGteLtLte(t *testing.T) {
	v := 5
	c := Gt("age", &v)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age > ?", sql)
	assert.Equal(t, []any{5}, args)

	c = Gte("age", &v)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age >= ?", sql)
	assert.Equal(t, []any{5}, args)

	c = Lt("age", &v)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age < ?", sql)
	assert.Equal(t, []any{5}, args)

	c = Lte("age", &v)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "age <= ?", sql)
	assert.Equal(t, []any{5}, args)
}

func TestLikes(t *testing.T) {
	c := Likes("name", []string{"a", "b"})
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "(name LIKE ? OR name LIKE ?)", sql)
	assert.Equal(t, []any{"a", "b"}, args)

	c = Likes("name", nil)
	assert.Nil(t, c)
}

func TestMultiLike(t *testing.T) {
	c := MultiLike([]string{"name", "desc"}, "foo")
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "(name LIKE ? OR desc LIKE ?)", sql)
	assert.Equal(t, []any{"foo", "foo"}, args)

	c = MultiLike([]string{"name", "desc"}, "  ")
	assert.Nil(t, c)
}

func TestNotIn(t *testing.T) {
	vals := []int{1, 2, 3}
	c := NotIn("id", &vals)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "id NOT IN (?,?,?)", sql)
	assert.Equal(t, []any{1, 2, 3}, args)

	c = NotIn("id", (*[]int)(nil))
	assert.Nil(t, c)
}

func TestTimeRangeBranches(t *testing.T) {
	// start nil, end set
	end := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	c := TimeRange("created_at", nil, &end)
	sql, args, err := c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "created_at < ?", sql)
	assert.Equal(t, []any{"2023-01-02 00:00:00"}, args)

	// start set, end nil
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	c = TimeRange("created_at", &start, nil)
	sql, args, err = c.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "created_at >= ?", sql)
	assert.Equal(t, []any{"2023-01-01 00:00:00"}, args)
}
