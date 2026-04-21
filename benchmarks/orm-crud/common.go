package ormcrud

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

//go:embed schema.sql
var sqliteSchemaSQL string

//go:embed mysql.sql
var mysqlSchemaSQL string

//go:embed postgres.sql
var postgresSchemaSQL string

var benchmarkCtx = context.Background()

var nonDatabaseName = regexp.MustCompile(`[^a-z0-9_]+`)

type benchInput struct {
	Name     string
	Alias    *string
	Age      int
	AgeP     *int
	Active   bool
	ActiveP  *bool
	Email    string
	Tags     benchmodel.IntSlice
	Meta     benchmodel.StringMap
	Profile  benchmodel.Profile
	Contacts benchmodel.ContactList
}

type benchmarkBackend struct {
	name       string
	sqlDriver  string
	lormDriver string
	entDialect string
}

type benchmarkDatabase struct {
	backend benchmarkBackend
	dsn     string
}

func makeBenchInput(i int) benchInput {
	alias := fmt.Sprintf("alias-%d", i)
	ageP := 100 + (i % 10)
	activeP := i%3 == 0
	return benchInput{
		Name:    fmt.Sprintf("user-%d", i),
		Alias:   &alias,
		Age:     18 + (i % 50),
		AgeP:    &ageP,
		Active:  i%2 == 0,
		ActiveP: &activeP,
		Email:   fmt.Sprintf("user-%d@example.com", i),
		Tags:    benchmodel.IntSlice{i, i + 1, i + 2},
		Meta: benchmodel.StringMap{
			"source": "benchmark",
			"key":    fmt.Sprintf("meta-%d", i),
		},
		Profile: benchmodel.Profile{
			ID:     i,
			Name:   fmt.Sprintf("profile-%d", i),
			Active: i%2 == 0,
			Labels: []string{fmt.Sprintf("l-%d", i), fmt.Sprintf("l-%d", i+1)},
		},
		Contacts: benchmodel.ContactList{
			{Kind: "email", Value: fmt.Sprintf("user-%d@example.com", i), Primary: true},
			{Kind: "phone", Value: fmt.Sprintf("1380000%04d", i%10000), Primary: false},
		},
	}
}

func benchmarkDSN(t testing.TB, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name+".sqlite")
}

func initSQLiteSchema(t testing.TB, dsn string) {
	t.Helper()

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	for _, stmt := range strings.Split(sqliteSchemaSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err = db.Exec(stmt); err != nil {
			t.Fatalf("init schema: %v", err)
		}
	}
}

func currentBackend(t testing.TB) benchmarkBackend {
	t.Helper()

	switch strings.ToLower(strings.TrimSpace(os.Getenv("ORMCRUD_DB"))) {
	case "", "sqlite":
		return benchmarkBackend{name: "sqlite", sqlDriver: "sqlite3", lormDriver: "sqlite3", entDialect: "sqlite"}
	case "mysql":
		return benchmarkBackend{name: "mysql", sqlDriver: "mysql", lormDriver: "mysql", entDialect: "mysql"}
	case "postgres", "pgsql", "postgresql":
		return benchmarkBackend{name: "postgres", sqlDriver: "pgx", lormDriver: "pgx", entDialect: "postgres"}
	default:
		t.Fatalf("unsupported ORMCRUD_DB %q", os.Getenv("ORMCRUD_DB"))
		return benchmarkBackend{}
	}
}

func prepareBenchmarkDatabase(t testing.TB, ormName, caseName string) benchmarkDatabase {
	t.Helper()

	backend := currentBackend(t)
	switch backend.name {
	case "sqlite":
		dsn := benchmarkDSN(t, ormName+"_"+caseName)
		initSQLiteSchema(t, dsn)
		return benchmarkDatabase{backend: backend, dsn: dsn}
	case "mysql":
		return prepareMySQLDatabase(t, backend, ormName, caseName)
	case "postgres":
		return preparePostgresDatabase(t, backend, ormName, caseName)
	default:
		t.Fatalf("unsupported backend %q", backend.name)
		return benchmarkDatabase{}
	}
}

func prepareMySQLDatabase(t testing.TB, backend benchmarkBackend, ormName, caseName string) benchmarkDatabase {
	t.Helper()

	dbName := benchmarkDatabaseName(backend.name, ormName, caseName)
	adminDSN := "root:123456@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true&charset=utf8mb4&loc=Local"
	adminDB, err := sql.Open(backend.sqlDriver, adminDSN)
	if err != nil {
		t.Fatalf("open mysql admin db: %v", err)
	}
	defer adminDB.Close()

	if _, err = adminDB.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create mysql database: %v", err)
	}
	t.Cleanup(func() {
		cleanupDB, err := sql.Open(backend.sqlDriver, adminDSN)
		if err != nil {
			t.Fatalf("open mysql admin db for cleanup: %v", err)
		}
		defer cleanupDB.Close()
		if _, err = cleanupDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`"); err != nil {
			t.Fatalf("drop mysql database: %v", err)
		}
	})

	dsn := fmt.Sprintf("root:123456@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true&charset=utf8mb4&loc=Local", dbName)
	initSQLSchema(t, backend.sqlDriver, dsn, mysqlSchemaSQL)
	return benchmarkDatabase{backend: backend, dsn: dsn}
}

func preparePostgresDatabase(t testing.TB, backend benchmarkBackend, ormName, caseName string) benchmarkDatabase {
	t.Helper()

	dbName := benchmarkDatabaseName(backend.name, ormName, caseName)
	adminDSN := "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=postgres sslmode=disable"
	adminDB, err := sql.Open(backend.sqlDriver, adminDSN)
	if err != nil {
		t.Fatalf("open postgres admin db: %v", err)
	}
	defer adminDB.Close()

	if _, err = adminDB.Exec(`CREATE DATABASE "` + dbName + `"`); err != nil {
		t.Fatalf("create postgres database: %v", err)
	}
	t.Cleanup(func() {
		cleanupDB, err := sql.Open(backend.sqlDriver, adminDSN)
		if err != nil {
			t.Fatalf("open postgres admin db for cleanup: %v", err)
		}
		defer cleanupDB.Close()
		if _, err = cleanupDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName); err != nil {
			t.Fatalf("terminate postgres connections: %v", err)
		}
		if _, err = cleanupDB.Exec(`DROP DATABASE IF EXISTS "` + dbName + `"`); err != nil {
			t.Fatalf("drop postgres database: %v", err)
		}
	})

	dsn := fmt.Sprintf("host=127.0.0.1 port=5432 user=postgres password=123456 dbname=%s sslmode=disable", dbName)
	initSQLSchema(t, backend.sqlDriver, dsn, postgresSchemaSQL)
	return benchmarkDatabase{backend: backend, dsn: dsn}
}

func initSQLSchema(t testing.TB, driverName, dsn, schema string) {
	t.Helper()

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("open %s db: %v", driverName, err)
	}
	defer db.Close()

	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err = db.Exec(stmt); err != nil {
			t.Fatalf("init %s schema: %v", driverName, err)
		}
	}
}

func benchmarkDatabaseName(backendName, ormName, caseName string) string {
	raw := strings.ToLower(fmt.Sprintf("ormcrud_%s_%s_%s_%d", backendName, ormName, caseName, time.Now().UnixNano()))
	return strings.Trim(nonDatabaseName.ReplaceAllString(raw, "_"), "_")
}

func nowForBench() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}
