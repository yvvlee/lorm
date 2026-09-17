package ormcrud

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

const sqlReadColumns = "id, name, alias, age, age_p, active, active_p, email, tags, meta, profile, contacts, created_at, updated_at"

func benchmarkReadByIDSQL(b *testing.B)            { benchmarkReadSQL(b, 1, false) }
func benchmarkReadByIDComplexSQL(b *testing.B)     { benchmarkReadSQL(b, 1, true) }
func benchmarkBatchRead100SQL(b *testing.B)        { benchmarkReadSQL(b, batchSize, false) }
func benchmarkBatchRead100ComplexSQL(b *testing.B) { benchmarkReadSQL(b, batchSize, true) }

// Seed data, connection setup and SQL preparation are outside the timer.
// Timed reads use database/sql directly with the same fields and JSON scanners
// as LormUser. This is an application-level baseline without ORM query building.
func benchmarkReadSQL(b *testing.B, count int, complex bool) {
	name := fmt.Sprintf("read_%d_%t", count, complex)
	config := prepareBenchmarkDatabase(b, "sql", name)
	db, err := sql.Open(config.backend.sqlDriver, config.dsn)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := db.PingContext(benchmarkCtx); err != nil {
		b.Fatal(err)
	}
	placeholders := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = "?"
			if config.backend.name == "postgres" {
				parts[i] = fmt.Sprintf("$%d", i+1)
			}
		}
		return strings.Join(parts, ",")
	}
	insertSQL := "INSERT INTO bench_users (" + sqlReadColumns + ") VALUES (" + placeholders(14) + ")"
	emails := make([]any, count)
	now := time.Unix(1700000000, 0).UTC()
	for i := range count {
		input := makeBenchInput(i)
		if complex {
			input = makeComplexBenchInput(i)
		}
		emails[i] = input.Email
		_, err := db.ExecContext(benchmarkCtx, insertSQL,
			int64(i+1), input.Name, input.Alias, input.Age, input.AgeP,
			input.Active, input.ActiveP, input.Email, input.Tags, input.Meta,
			input.Profile, input.Contacts, now, now,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
	query := "SELECT " + sqlReadColumns + " FROM bench_users WHERE email IN (" + placeholders(count) + ")"
	args := emails
	if count == 1 {
		query = "SELECT " + sqlReadColumns + " FROM bench_users WHERE id = " + placeholders(1) + " LIMIT 1"
		args = []any{int64(1)}
	}
	b.ReportAllocs()
	for b.Loop() {
		if count == 1 {
			out, err := readSQLUser(db, query, args)
			if err != nil {
				b.Fatal(err)
			}
			if out.ID != 1 || out.Name == "" {
				b.Fatal("unexpected SQL read result")
			}
			if complex {
				consumeLormUser(out)
			}
			continue
		}
		out, err := readSQLUsers(db, query, args)
		if err != nil {
			b.Fatal(err)
		}
		if len(out) != count || out[0].Name == "" {
			b.Fatal("unexpected SQL read result")
		}
		if complex {
			consumeLormUsers(out)
		}
	}
}

func readSQLUsers(db *sql.DB, query string, args []any) ([]*LormUser, error) {
	rows, err := db.QueryContext(benchmarkCtx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*LormUser
	for rows.Next() {
		u := new(LormUser)
		if err := rows.Scan(&u.ID, &u.Name, &u.Alias, &u.Age, &u.AgeP,
			&u.Active, &u.ActiveP, &u.Email, &u.Tags, &u.Meta,
			&u.Profile, &u.Contacts, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return result, nil
}

// Match Get's single-row result ownership without allocating a result slice.
func readSQLUser(db *sql.DB, query string, args []any) (*LormUser, error) {
	rows, err := db.QueryContext(benchmarkCtx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	u := new(LormUser)
	if err := rows.Scan(&u.ID, &u.Name, &u.Alias, &u.Age, &u.AgeP,
		&u.Active, &u.ActiveP, &u.Email, &u.Tags, &u.Meta,
		&u.Profile, &u.Contacts, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return u, nil
}
