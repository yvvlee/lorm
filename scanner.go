package lorm

import (
	"database/sql"
	"fmt"
)

// ScanModels scans all rows into models using their descriptor field mapping.
func ScanModels[T Model](rows *sql.Rows, models *[]T) error {
	res, err := scanModelValues[T](rows)
	if err != nil {
		return err
	}
	*models = res
	return nil
}

func scanModelValues[T Model](rows *sql.Rows) ([]T, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
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
			return nil, err
		}
		typed, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("lorm: Model.New returned %T, want %T", item, model)
		}
		res = append(res, typed)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func scanModelValue[T Model](rows *sql.Rows) (T, error) {
	var value T
	item := value.New()
	if err := ScanModel(rows, item); err != nil {
		return value, err
	}
	typed, ok := item.(T)
	if !ok {
		return value, fmt.Errorf("lorm: Model.New returned %T, want %T", item, value)
	}
	return typed, nil
}

func scanOrderedModelValues[T Model](rows *sql.Rows) ([]T, error) {
	var model T
	if _, ok := any(model).(orderedModelScanner); !ok {
		return scanModelValues[T](rows)
	}
	var res []T
	for rows.Next() {
		item := model.New()
		scanner, ok := item.(orderedModelScanner)
		if !ok {
			return nil, fmt.Errorf("lorm: Model.New returned %T without generated ordered scanner", item)
		}
		if err := scanner.LormScan(rows); err != nil {
			return nil, err
		}
		typed, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("lorm: Model.New returned %T, want %T", item, model)
		}
		res = append(res, typed)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func scanOrderedModelValue[T Model](rows *sql.Rows) (T, error) {
	var value T
	item := value.New()
	scanner, ok := item.(orderedModelScanner)
	if !ok {
		return scanModelValue[T](rows)
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return value, err
		}
		return value, sql.ErrNoRows
	}
	if err := scanner.LormScan(rows); err != nil {
		return value, err
	}
	typed, ok := item.(T)
	if !ok {
		return value, fmt.Errorf("lorm: Model.New returned %T, want %T", item, value)
	}
	return typed, nil
}

func scanColumnValues[T any](rows *sql.Rows) ([]T, error) {
	var values []T
	if err := ScanCols(rows, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// ScanCols scans the only column of each row into v.
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

// ScanCol scans the only column of the first row into t.
func ScanCol[T any](row *sql.Rows, t *T) error {
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
