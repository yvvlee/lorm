package lorm

import (
	"database/sql"
	"fmt"
)

// ScanModels scans all rows into models using their descriptor field mapping.
func ScanModels[T Model](rows *sql.Rows, models *[]T) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	var res []T
	var model T
	for rows.Next() {
		item := model.New()
		values := make([]any, len(columns))
		for i, column := range columns {
			field := item.LormFieldPtr(column)
			if field == nil {
				// Keep scanning aligned even when the model does not expose this column.
				values[i] = new(any)
				continue
			}
			values[i] = field
		}
		if err = rows.Scan(values...); err != nil {
			return err
		}
		res = append(res, item.(T))
	}
	// Check if there was an error during iteration
	if err = rows.Err(); err != nil {
		return err
	}
	*models = res
	return nil
}

// ScanCols scans the first column of each row into v.
func ScanCols[T any](rows *sql.Rows, v *[]T) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) != 1 {
		return fmt.Errorf("expected exactly one column, got %d", len(columns))
	}
	if len(*v) > 0 {
		return fmt.Errorf("ScanCols requires an empty destination slice, got len=%d", len(*v))
	}
	var res []T
	for rows.Next() {
		var item T
		if err = rows.Scan(&item); err != nil {
			return err
		}
		res = append(res, item)
	}
	*v = res
	return rows.Err()
}

// ScanModel scans the first row into m using column names to locate fields.
func ScanModel[T Model](row *sql.Rows, m T) error {
	columns, err := row.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	for i, column := range columns {
		field := m.LormFieldPtr(column)
		if field == nil {
			// Keep scanning aligned even when the model does not expose this column.
			values[i] = new(any)
			continue
		}
		values[i] = field
	}
	return scanRow(row, values...)
}

// ScanCol scans the first column of the first row into t.
// t must be a pointer (e.g. *int, *string) so the scanned value
// can be written back to the caller.
func ScanCol[T any](row *sql.Rows, t T) error {
	columns, err := row.Columns()
	if err != nil {
		return err
	}
	if len(columns) != 1 {
		return fmt.Errorf("expected exactly one column, got %d", len(columns))
	}
	return scanRow(row, t)
}

func scanRow(rows *sql.Rows, dest ...any) error {
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dest...)
}
