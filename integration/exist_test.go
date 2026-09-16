package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm"
)

func TestExistPreservesQueryResults(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.Background()
	insertBasicRows(t, e, ctx)
	for _, tt := range []struct {
		name  string
		build func(*lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test]
		want  bool
	}{
		{"simple hit", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] { return s.Where("id > ?", 0) }, true},
		{"simple miss", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] { return s.Where("id < ?", 0) }, false},
		{"default distinct offset", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] { return s.Distinct().Limit(10).Offset(1) }, true},
		{"distinct offset", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("str").Distinct().Limit(10).Offset(1)
		}, true},
		{"distinct exhausted", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("str").Distinct().Limit(10).Offset(2)
		}, false},
		{"aggregate having", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("COUNT(*) AS total").Having("COUNT(*) > ?", 1)
		}, true},
		{"aggregate having miss", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("COUNT(*) AS total").Having("COUNT(*) > ?", 2)
		}, false},
		{"empty aggregate still has a row", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("COUNT(*)").Where("id < ?", 0)
		}, true},
		{"zero limit", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] { return s.Limit(0) }, false},
		{"group having", func(s *lorm.SelectStmt[*Test]) *lorm.SelectStmt[*Test] {
			return s.Select("str").GroupBy("str").Having("COUNT(*) > ?", 1)
		}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stmt := tt.build(e.Query[*Test]())
			exists, err := stmt.Exist(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.want, exists)
			// Terminal calls reset projection and pagination as well as conditions.
			count, err := stmt.Count(ctx)
			require.NoError(t, err)
			assert.EqualValues(t, 2, count)
		})
	}
}

func TestExistReturnsPostgresErrorAfterFirstRow(t *testing.T) {
	e := initEngine(t)
	t.Cleanup(func() { _ = e.Close() })
	if e.DriverName() != "pgx" {
		t.Skip("PostgreSQL reports this execution error while closing the result set")
	}
	exists, err := e.Exist(context.Background(), "SELECT 1 / (2 - x) FROM generate_series(1, 2) AS x")
	require.ErrorContains(t, err, "division by zero")
	require.False(t, exists)
}
