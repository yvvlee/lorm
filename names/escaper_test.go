package names

import (
	"testing"
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
