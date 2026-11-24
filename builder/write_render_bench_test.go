package builder

import "testing"

func BenchmarkWriteRender(b *testing.B) {
	insert := Insert("users").Columns("id", "name", "age")
	for i := 0; i < 100; i++ {
		insert.Values(i, "Alice", 30)
	}
	b.Run("insert100", func(b *testing.B) { benchmarkWrite(b, insert) })
	b.Run("update", func(b *testing.B) {
		benchmarkWrite(b, Update("users").Set("name", "Alice").Set("age", 30).Where("id = ?", 1))
	})
	b.Run("delete", func(b *testing.B) {
		benchmarkWrite(b, Delete("users").Where("id = ?", 1))
	})
}

func benchmarkWrite(b *testing.B, query Sqlizer) {
	b.Helper()
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := query.ToSql()
		if err != nil {
			b.Fatal(err)
		}
	}
}
