package lorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Session struct {
	engine   *Engine
	tx       *sql.Tx
	isClosed bool
}

func (s *Session) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	proxy := s.proxy()
	if len(args) == 0 {
		return proxy.ExecContext(ctx, query)
	}
	var err error
	query, err = s.engine.Placeholder().ReplacePlaceholders(query)
	if err != nil {
		return nil, err
	}
	stmt, err := proxy.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.ExecContext(ctx, args...)
}

func (s *Session) Query(ctx context.Context, scanner Scanner, query string, args ...any) (err error) {
	proxy := s.proxy()
	var rows *sql.Rows
	if len(args) == 0 {
		rows, err = proxy.QueryContext(ctx, query)
	} else {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return err
		}
		var stmt *sql.Stmt
		stmt, err = proxy.PrepareContext(ctx, query)
		if err != nil {
			return err
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanner.Scan(rows)
}

func (s *Session) Exist(ctx context.Context, query string, args ...any) (exist bool, err error) {
	proxy := s.proxy()
	var rows *sql.Rows
	if len(args) == 0 {
		rows, err = proxy.QueryContext(ctx, query)
	} else {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
		var stmt *sql.Stmt
		stmt, err = proxy.PrepareContext(ctx, query)
		if err != nil {
			return
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args...)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		return true, nil
	}
	return false, rows.Err()
}

func (s *Session) proxy() DBProxy {
	if s.tx != nil {
		return s.tx
	}
	return s.engine.db
}

func (s *Session) close() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	if s.tx == nil {
		return nil
	}
	return s.tx.Rollback()
}
func (s *Session) commit() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	if s.tx == nil {
		return nil
	}
	return s.tx.Commit()
}

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
				return fmt.Errorf("未找到映射字段：%s", column)
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
	// 检查迭代过程是否有错误
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
			return fmt.Errorf("未找到映射字段：%s", column)
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

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Preparer interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type DBProxy interface {
	Execer
	Querier
	Preparer
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
