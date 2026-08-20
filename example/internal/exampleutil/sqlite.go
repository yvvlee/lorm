package exampleutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/yvvlee/lorm"
)

var ErrSQLiteDriverUnavailable = errors.New("sqlite3 driver unavailable")

// IsSQLiteDriverUnavailable reports the known go-sqlite3 CGO-disabled failure.
func IsSQLiteDriverUnavailable(err error) bool {
	return errors.Is(err, ErrSQLiteDriverUnavailable)
}

// NewSQLiteEngine creates a temporary SQLite database, initializes schema, and
// returns an Engine plus a cleanup function.
func NewSQLiteEngine(schema string) (*lorm.Engine, func(), error) {
	tempDir, err := os.MkdirTemp("", "lorm-example-*")
	if err != nil {
		return nil, nil, err
	}

	dbPath := filepath.Join(tempDir, "example.sqlite")
	engine, err := lorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			return nil, nil, fmt.Errorf("%w: %v", ErrSQLiteDriverUnavailable, err)
		}
		return nil, nil, err
	}

	cleanup := func() {
		_ = engine.Close()
		_ = os.RemoveAll(tempDir)
	}

	ctx := context.Background()
	if _, err := engine.Exec(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := engine.Exec(ctx, stmt); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("init schema: %w", err)
		}
	}

	return engine, cleanup, nil
}
