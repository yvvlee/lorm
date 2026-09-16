package names

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoter_Escape(t *testing.T) {
	tests := []struct {
		name         string
		prefix       byte
		suffix       byte
		fieldOrTable string
		expected     string
	}{
		{
			name:         "empty string",
			prefix:       '`',
			suffix:       '`',
			fieldOrTable: "",
			expected:     "",
		},
		{
			name:         "simple field",
			prefix:       '`',
			suffix:       '`',
			fieldOrTable: "field",
			expected:     "`field`",
		},
		{
			name:         "field with prefix and suffix already",
			prefix:       '`',
			suffix:       '`',
			fieldOrTable: "`field`",
			expected:     "`field`",
		},
		{
			name:         "table.field format",
			prefix:       '`',
			suffix:       '`',
			fieldOrTable: "table.field",
			expected:     "`table`.`field`",
		},
		{
			name:         "table.field with mixed quotes",
			prefix:       '`',
			suffix:       '`',
			fieldOrTable: "`table`.field",
			expected:     "`table`.`field`",
		},
		{
			name:         "no escaper",
			prefix:       0,
			suffix:       0,
			fieldOrTable: "table.field",
			expected:     "table.field",
		},
		{
			name:         "different prefix and suffix",
			prefix:       '[',
			suffix:       ']',
			fieldOrTable: "table.field",
			expected:     "[table].[field]",
		},
		{
			name:         "escape internal quote characters",
			prefix:       '"',
			suffix:       '"',
			fieldOrTable: `schema.a"b`,
			expected:     `"schema"."a""b"`,
		},
		{
			name:         "escape internal suffix characters with asymmetric quotes",
			prefix:       '[',
			suffix:       ']',
			fieldOrTable: `schema.a]b`,
			expected:     `[schema].[a]]b]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQuoter(tt.prefix, tt.suffix)
			result := q.Escape(tt.fieldOrTable)
			if result != tt.expected {
				t.Errorf("Escape(%q) = %q, want %q", tt.fieldOrTable, result, tt.expected)
			}
		})
	}
}

func TestQuoterPreservesQuotedIdentifierBoundaries(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{`"a.b"`, `"a.b"`},
		{`schema."a.b"`, `"schema"."a.b"`},
		{`"schema.name".field`, `"schema.name"."field"`},
		{`"schema.name"."a"".b"`, `"schema.name"."a"".b"`},
		{`a"b.field`, `"a""b"."field"`},
	} {
		q := NewQuoter('"', '"')
		assert.Equal(t, tt.want, q.Escape(tt.input))
		assert.Equal(t, tt.want, q.Escape(q.Escape(tt.input)))
	}
	q := NewQuoter('[', ']')
	assert.Equal(t, `[schema.name].[a]].b]`, q.Escape(`[schema.name].[a]].b]`))
}

func TestQuoterRejectsMalformedQuotedIdentifiers(t *testing.T) {
	q := NewQuoter('"', '"')
	for _, input := range []string{`"value" DESC, "id"`, `"unclosed`, `schema."field" ASC`, `"field"suffix`, `"a""`} {
		assert.Panics(t, func() { q.Escape(input) }, input)
	}
}

func TestNoEscaper(t *testing.T) {
	result := NoEscaper.Escape("test")
	if result != "test" {
		t.Errorf("NoEscaper.Escape('test') = %q, want ''", result)
	}

	result = NoEscaper.Escape("")
	if result != "" {
		t.Errorf("NoEscaper.Escape('') = %q, want ''", result)
	}
}
