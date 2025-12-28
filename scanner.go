package lorm

import (
	"database/sql"
	"errors"
	"fmt"
)

func ScanModels[T Model](rows *sql.Rows, models *[]T) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	var res []T
	var model T
	jsonFields := make(map[string]struct{})
	for _, field := range model.LormModelDescriptor().Fields {
		if field.Flag.HasFlag(FlagJson) {
			jsonFields[field.DBField] = struct{}{}
		}
	}
	for rows.Next() {
		item := model.New()
		fieldsMap := item.LormFieldMap()
		values := make([]any, len(columns))
		for i, column := range columns {
			field, ok := fieldsMap[column]
			if !ok {
				values[i] = new(sql.RawBytes)
				continue
			}
			if _, ok = jsonFields[column]; ok {
				field = NewJSONFieldWrapper(field)
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

func ScanCols[T any](rows *sql.Rows, v *[]T) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) != 1 {
		return fmt.Errorf("expected exactly one column, got %d", len(columns))
	}
	if len(*v) > 0 {
		i := 0
		for rows.Next() && i < len(*v) {
			item := (*v)[i]
			if err = rows.Scan(item); err != nil {
				return err
			}
			i++
		}
		return nil
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
	return nil
}

func ScanModel[T Model](row *sql.Rows, m T) error {
	columns, err := row.Columns()
	if err != nil {
		return err
	}
	jsonFields := make(map[string]struct{})
	for _, field := range m.LormModelDescriptor().Fields {
		if field.Flag.HasFlag(FlagJson) {
			jsonFields[field.DBField] = struct{}{}
		}
	}
	fieldsMap := m.LormFieldMap()
	values := make([]any, len(columns))
	for i, column := range columns {
		field, ok := fieldsMap[column]
		if !ok {
			values[i] = new(sql.RawBytes)
			continue
		}
		if _, ok = jsonFields[column]; ok {
			field = NewJSONFieldWrapper(field)
		}
		values[i] = field
	}
	return scanRow(row, values...)
}

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
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on Row.Scan")
		}
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dest...)
}
