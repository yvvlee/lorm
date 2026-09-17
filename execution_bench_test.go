package lorm

import (
	"database/sql"
	"database/sql/driver"
	"strconv"
	"testing"
	"time"

	"github.com/yvvlee/lorm/builder"
)

// Includes statement construction, identifier quoting, cached default projection,
// SQL rendering and the configured placeholder conversion; no database I/O.
func BenchmarkSelectSQL(b *testing.B) {
	for _, dialect := range []string{"mysql", "pgx"} {
		b.Run(dialect, func(b *testing.B) {
			engine := &Engine{config: &Config{Dialect: DefaultDialectConfig(dialect)}}
			if _, err := engine.defaultSelectProjection((*Test)(nil).LormModelDescriptor()); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				query, args, err := engine.Query[*Test]().
					Where(builder.Gte("id", 1)).Asc("id").Limit(100).ToSql()
				if err != nil {
					b.Fatal(err)
				}
				if query == "" || len(args) != 1 {
					b.Fatal("unexpected SELECT")
				}
			}
		})
	}
}

// Uses the generated ordered scanner with nullable, decimal and JSON fields.
// The scripted database isolates scanning costs from server and network latency.
func BenchmarkScanModelsDefaultColumns(b *testing.B) {
	for _, count := range []int{1, 100, 1000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			r := newScriptedQueryRecorder()
			db, err := sql.Open(registerScriptedQueryDriver(r), "")
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = db.Close() })
			now := time.Unix(1700000000, 0)
			result := scriptedQueryResult{columns: testInsertColumns}
			for i := range count {
				result.rows = append(result.rows, []driver.Value{
					int64(i + 1), int64(2), nil, true, nil, "name", nil,
					now, nil, now, nil, "1.25", nil, "[1,2]", nil,
					`{"id":1,"name":"profile"}`, nil, now, now,
				})
			}
			b.ReportAllocs()
			for b.Loop() {
				r.results = append(r.results[:0], result)
				r.queryCalls = r.queryCalls[:0]
				rows, err := db.Query("SELECT model_columns FROM test")
				if err != nil {
					b.Fatal(err)
				}
				models, scanErr := scanOrderedModelValues[*Test](rows)
				closeErr := rows.Close()
				if scanErr != nil {
					b.Fatal(scanErr)
				}
				if closeErr != nil {
					b.Fatal(closeErr)
				}
				if len(models) != count || models[count-1].ID != uint64(count) || len(models[0].IntSlice) != 2 {
					b.Fatal("unexpected scanned models")
				}
			}
		})
	}
}

// Measures plan allocation and shape validation, excluding SQL rendering and I/O.
func BenchmarkInsertPlanBatch(b *testing.B) {
	for _, count := range []int{1, 100, 1000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			models := make([]*Test, count)
			for i := range models {
				models[i] = new(Test)
			}
			now := time.Unix(1700000000, 0)
			b.ReportAllocs()
			for b.Loop() {
				plans := make([]InsertPlan, count)
				for i, model := range models {
					*model = Test{Str: "name"}
					plan, err := prepareInsertPlan(model, now, nil)
					if err != nil {
						b.Fatal(err)
					}
					if err := validateInsertPlan(plan, i); err != nil {
						b.Fatal(err)
					}
					plans[i] = plan
					if i > 0 && !sameInsertShape(plans[0], plan) {
						b.Fatal("inconsistent insert shape")
					}
				}
				if len(plans[count-1].Values) != len(testInsertColumnsWithoutAutoIncrement) {
					b.Fatal("unexpected insert plan")
				}
			}
		})
	}
}
