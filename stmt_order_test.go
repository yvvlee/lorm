package lorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestStatementOrdering(t *testing.T) {
	for _, dialect := range []struct {
		name    string
		escaper names.Escaper
		want    string
	}{
		{"unquoted", names.NoEscaper, "group ASC, u.id ASC, created_at DESC, id DESC, COALESCE(score, 0) DESC, name ASC"},
		{"mysql", names.NewQuoter('`', '`'), "`group` ASC, `u`.`id` ASC, `created_at` DESC, `id` DESC, COALESCE(score, 0) DESC, `name` ASC"},
		{"postgres", names.NewQuoter('"', '"'), "\"group\" ASC, \"u\".\"id\" ASC, \"created_at\" DESC, \"id\" DESC, COALESCE(score, 0) DESC, \"name\" ASC"},
	} {
		t.Run(dialect.name, func(t *testing.T) {
			engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: dialect.escaper}}}
			columns := []string{"group", "u.id"}
			selectStmt := engine.Query[*Test]().Select("id").Asc().Desc().
				Asc(columns...).Desc("created_at", "id").OrderBy("COALESCE(score, 0) DESC").Asc("name")
			updateStmt := engine.Update[*Test]().Set("name", "new").Asc().Desc().
				Asc(columns...).Desc("created_at", "id").OrderBy("COALESCE(score, 0) DESC").Asc("name")
			deleteStmt := engine.Delete[*Test]().Asc().Desc().
				Asc(columns...).Desc("created_at", "id").OrderBy("COALESCE(score, 0) DESC").Asc("name")
			for name, stmt := range map[string]builder.Sqlizer{
				"select": selectStmt.builder,
				"update": updateStmt.builder,
				"delete": deleteStmt.builder,
			} {
				t.Run(name, func(t *testing.T) {
					query, _, err := stmt.ToSql()
					require.NoError(t, err)
					assert.Contains(t, query, " ORDER BY "+dialect.want)
				})
			}
			assert.Equal(t, []string{"group", "u.id"}, columns)
		})
	}
}
