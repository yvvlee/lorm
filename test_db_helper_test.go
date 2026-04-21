package lorm

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
)

var (
	sqlite3TestCheckOnce sync.Once
	sqlite3TestCheckErr  error
)

func sqlite3TestAvailable() (bool, string) {
	sqlite3TestCheckOnce.Do(func() {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			sqlite3TestCheckErr = err
			return
		}
		defer db.Close()
		if err = db.Ping(); err != nil {
			sqlite3TestCheckErr = err
			return
		}
		if _, err = db.Exec("SELECT 1"); err != nil {
			sqlite3TestCheckErr = err
		}
	})
	if sqlite3TestCheckErr != nil {
		return false, sqlite3TestCheckErr.Error()
	}
	return true, ""
}

func skipUnlessSQLite3Available(t *testing.T) {
	t.Helper()
	if ok, reason := sqlite3TestAvailable(); !ok {
		t.Skipf("sqlite3 test dependency unavailable: %s", reason)
	}
}

func integrationTestDriverAndDSN(t *testing.T) (string, string) {
	t.Helper()
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")
	if driver != "" && dsn != "" {
		return driver, dsn
	}
	if ok, _ := sqlite3TestAvailable(); ok {
		return "sqlite3", "file:lorm_test.db?cache=shared&mode=memory"
	}
	t.Skipf("integration tests require DB_DRIVER and DB_DSN, or a working sqlite3 test driver")
	return "", ""
}

func mustIntegrationDriverAndDSN(t *testing.T) (string, string) {
	t.Helper()
	driver, dsn := integrationTestDriverAndDSN(t)
	if driver == "" || dsn == "" {
		t.Fatal("missing integration test driver or DSN")
	}
	return driver, dsn
}

func initIntegrationSQL(driver string) (string, error) {
	switch driver {
	case "postgres", "pgx":
		return postgresInitSQL, nil
	case "sqlite3", "sqlite":
		return sqliteInitSQL, nil
	case "mysql", "mariadb":
		return mysqlInitSQL, nil
	default:
		return "", fmt.Errorf("unsupported integration test driver %q", driver)
	}
}
