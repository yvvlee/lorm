package lorm

import (
	"database/sql"
	"errors"
	"fmt"
)

type Scanner interface {
	Scan(*sql.Rows) error
}

type ModelsScanner[T Model] struct {
	models *[]T
}

func NewModelsScanner[T Model](models *[]T) *ModelsScanner[T] {
	return &ModelsScanner[T]{models: models}
}
func (m *ModelsScanner[T]) Scan(rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	var models []T
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
		models = append(models, item.(T))
	}
	// Check if there was an error during iteration
	if err = rows.Err(); err != nil {
		return err
	}
	*m.models = models
	return nil
}

type ColsScanner[T any] struct {
	v *[]T
}

func NewColsScanner[T any](v *[]T) *ColsScanner[T] {
	return &ColsScanner[T]{v: v}
}

func (m *ColsScanner[T]) Scan(rows *sql.Rows) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) != 1 {
		return fmt.Errorf("expected exactly one column, got %d", len(columns))
	}
	var v []T
	for rows.Next() {
		var item T
		if err = rows.Scan(&item); err != nil {
			return err
		}
		v = append(v, item)
	}
	*m.v = v
	return nil
}

type ModelScanner[T Model] struct {
	model T
}

func NewModelScanner[T Model](model T) *ModelScanner[T] {
	return &ModelScanner[T]{model: model}
}

func (m *ModelScanner[T]) Scan(row *sql.Rows) error {
	columns, err := row.Columns()
	if err != nil {
		return err
	}
	jsonFields := make(map[string]struct{})
	for _, field := range m.model.LormModelDescriptor().Fields {
		if field.Flag.HasFlag(FlagJson) {
			jsonFields[field.DBField] = struct{}{}
		}
	}
	fieldsMap := m.model.LormFieldMap()
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

type ColScanner[T any] struct {
	v T
}

func NewColScanner[T any](v T) *ColScanner[T] {
	return &ColScanner[T]{v: v}
}

func (m *ColScanner[T]) Scan(row *sql.Rows) error {
	columns, err := row.Columns()
	if err != nil {
		return err
	}
	if len(columns) != 1 {
		return fmt.Errorf("expected exactly one column, got %d", len(columns))
	}
	return scanRow(row, m.v)
}

func scanRow(rows *sql.Rows, dest ...interface{}) error {
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
