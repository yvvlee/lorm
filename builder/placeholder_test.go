package builder

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuestion(t *testing.T) {
	sql := "x = ? AND y = ?"
	s, _ := Question.ReplacePlaceholders(sql)
	assert.Equal(t, sql, s)
}

func TestDollar(t *testing.T) {
	sql := "x = ? AND y = ?"
	s, _ := Dollar.ReplacePlaceholders(sql)
	assert.Equal(t, "x = $1 AND y = $2", s)
}

func TestColon(t *testing.T) {
	sql := "x = ? AND y = ?"
	s, _ := Colon.ReplacePlaceholders(sql)
	assert.Equal(t, "x = :1 AND y = :2", s)
}

func TestAtp(t *testing.T) {
	sql := "x = ? AND y = ?"
	s, _ := AtP.ReplacePlaceholders(sql)
	assert.Equal(t, "x = @p1 AND y = @p2", s)
}

func TestPlaceholders(t *testing.T) {
	assert.Equal(t, Placeholders(2), "?,?")
	assert.Equal(t, "", Placeholders(0))
}

func TestPlaceholderString(t *testing.T) {
	assert.Equal(t, "?", Question.PlaceholderString())
	assert.Equal(t, "$", Dollar.PlaceholderString())
	assert.Equal(t, ":", Colon.PlaceholderString())
	assert.Equal(t, "@p", AtP.PlaceholderString())
}

func TestEscapeDollar(t *testing.T) {
	sql := "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ??| array['?'] AND enabled = ?"
	s, _ := Dollar.ReplacePlaceholders(sql)
	assert.Equal(t, "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ?| array['$1'] AND enabled = $2", s)
}

func TestEscapeColon(t *testing.T) {
	sql := "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ??| array['?'] AND enabled = ?"
	s, _ := Colon.ReplacePlaceholders(sql)
	assert.Equal(t, "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ?| array[':1'] AND enabled = :2", s)
}

func TestEscapeAtp(t *testing.T) {
	sql := "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ??| array['?'] AND enabled = ?"
	s, _ := AtP.ReplacePlaceholders(sql)
	assert.Equal(t, "SELECT uuid, \"data\" #> '{tags}' AS tags FROM nodes WHERE  \"data\" -> 'tags' ?| array['@p1'] AND enabled = @p2", s)
}

func TestPositionalPlaceholderNumbering(t *testing.T) {
	for _, format := range []PlaceholderFormat{Dollar, Colon, AtP} {
		for _, count := range []int{99, 100, 128, 129, 1000} {
			t.Run(format.PlaceholderString()+"/"+strconv.Itoa(count), func(t *testing.T) {
				placeholders := make([]string, count)
				for i := range placeholders {
					placeholders[i] = format.PlaceholderString() + strconv.Itoa(i+1)
				}
				// Escaped question marks must not consume a sequence number.
				query := "SELECT ??, " + Placeholders(count) + ", ??"
				got, err := format.ReplacePlaceholders(query)
				assert.NoError(t, err)
				assert.Equal(t, "SELECT ?, "+strings.Join(placeholders, ",")+", ?", got)
			})
		}
	}
}

func BenchmarkPlaceholdersArray(b *testing.B) {
	for _, count := range []int{1, 10, 100, 1000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				placeholders := make([]string, count)
				for i := range placeholders {
					placeholders[i] = "?"
				}
				result := strings.Join(placeholders, ",")
				if len(result) != 2*count-1 {
					b.Fatal("unexpected placeholder length")
				}
			}
		})
	}
}

func BenchmarkPlaceholdersStrings(b *testing.B) {
	for _, count := range []int{1, 10, 100, 1000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result := Placeholders(count)
				if len(result) != 2*count-1 {
					b.Fatal("unexpected placeholder length")
				}
			}
		})
	}
}
