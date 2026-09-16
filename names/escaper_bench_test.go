package names

import "testing"

var escapedIdentifier string

func BenchmarkQuoterEscape(b *testing.B) {
	q := NewQuoter('"', '"')
	for _, tt := range []struct{ name, input string }{
		{"field", "created_at"},
		{"quoted", `"created_at"`},
		{"qualified", "users.created_at"},
		{"qualified_quoted", `"users"."created_at"`},
		{"embedded_quote", `user"name`},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				escapedIdentifier = q.Escape(tt.input)
			}
		})
	}
}
