package builder

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Keep input construction outside the timer to isolate rendering costs.
func BenchmarkSelectRender(b *testing.B) {
	columns := make([]string, 20)
	for i := range columns {
		columns[i] = fmt.Sprintf("column_%d", i)
	}
	ids := make([]int, 1000)
	for i := range ids {
		ids[i] = i
	}
	for _, tc := range []struct {
		name  string
		query *SelectBuilder
	}{
		{"simple", Select("id", "name").From("users").Where(Eq{"id": 1}).Limit(1)},
		{"prepared", new(SelectBuilder).SelectPrepared(NewPreparedProjection(strings.Join(columns, ", "))).
			From("users").Where(Gte("id", 1)).OrderBy("id").Limit(100)},
		{"complex", Select("u.id", "u.name", "COUNT(o.id) AS orders").From("users u").
			LeftJoin("orders o ON o.user_id = u.id").Where(Eq{"u.active": true}).
			GroupBy("u.id", "u.name").Having("COUNT(o.id) > ?", 3).
			OrderBy("orders DESC", "u.id").Limit(50).Offset(100)},
		{"large_in", Select("id", "name").From("users").Where(In("id", ids)).OrderBy("id").Limit(1000)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			want, wantArgs, err := tc.query.ToSql()
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				query, args, err := tc.query.ToSql()
				if err != nil {
					b.Fatal(err)
				}
				if len(query) != len(want) || len(args) != len(wantArgs) {
					b.Fatal("unexpected rendered SQL")
				}
			}
		})
	}
}

func BenchmarkReplacePlaceholders(b *testing.B) {
	for _, count := range []int{0, 1, 10, 99, 100, 128, 129, 1000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			query := "SELECT id FROM users"
			if count > 0 {
				query += " WHERE id IN (" + Placeholders(count) + ")"
			}
			want, err := Dollar.ReplacePlaceholders(query)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				got, err := Dollar.ReplacePlaceholders(query)
				if err != nil {
					b.Fatal(err)
				}
				if len(got) != len(want) {
					b.Fatal("unexpected placeholder conversion")
				}
			}
		})
	}
}
