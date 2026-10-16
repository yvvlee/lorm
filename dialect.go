package lorm

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

func detectDialectConfig(ctx context.Context, db *sql.DB, driverName string) (DialectConfig, error) {
	dialect := DefaultDialectConfig(driverName)
	if driverName != "sqlite" && driverName != "sqlite3" {
		return dialect, nil
	}

	// Query the linked library: a driver release can use a different system SQLite.
	var version string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		return DialectConfig{}, fmt.Errorf("lorm: detect SQLite version: %w", err)
	}
	supported, err := sqliteSupportsReturning(version)
	if err != nil {
		return DialectConfig{}, err
	}
	dialect.SupportsReturning = supported
	return dialect, nil
}

func sqliteSupportsReturning(version string) (bool, error) {
	parts := strings.Split(version, ".")
	// Historical SQLite versions used two to four components, e.g. 3.8.10.2.
	if len(parts) < 2 || len(parts) > 4 {
		return false, fmt.Errorf("lorm: invalid SQLite version %q: expected two to four numeric components", version)
	}
	var numbers [4]uint64
	for i, part := range parts {
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return false, fmt.Errorf("lorm: invalid SQLite version %q: %w", version, err)
		}
		numbers[i] = value
	}
	return numbers[0] > 3 || numbers[0] == 3 && numbers[1] >= 35, nil
}
