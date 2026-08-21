package lorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	json "github.com/bytedance/sonic"
)

// Table is a model that also exposes a database table name.
type Table interface {
	mustEmbedUnimplementedTable()
	TableName
	Model
}

// TableName lets a model override its database table name.
type TableName interface {
	TableName() string
}

// Model describes the metadata and field access lorm needs for persistence.
type Model interface {
	mustEmbedUnimplementedModel()
	// New create a new model instance
	New() Model
	// LormFieldPtr returns a field pointer by db field name. JSON fields should
	// return a JSONFieldWrapper around the underlying field pointer.
	LormFieldPtr(name string) any
	// LormModelDescriptor returns immutable metadata for this model. Repeated
	// calls must return the same descriptor pointer.
	LormModelDescriptor() *ModelDescriptor
}

// ModelPointer constrains a model to a pointer to its concrete struct type.
type ModelPointer[M any] interface {
	Model
	*M
}

// ModelFieldValueAccessor is implemented by generated models to expose field
// values to write execution and extension code without reflection.
type ModelFieldValueAccessor interface {
	LormFieldValue(name string) any
}

// RowScanner is the minimal row-scanning operation used by generated models.
type RowScanner interface {
	Scan(dest ...any) error
}

type orderedModelScanner interface {
	LormScan(row RowScanner) error
}

// UnimplementedModel can be embedded to satisfy the Model marker method.
type UnimplementedModel struct{}

func (u UnimplementedModel) mustEmbedUnimplementedModel() {}

// UnimplementedTable can be embedded to satisfy the Table marker methods.
type UnimplementedTable struct{}

func (u UnimplementedTable) mustEmbedUnimplementedModel() {}
func (u UnimplementedTable) mustEmbedUnimplementedTable() {}

// JSONFieldWrapper adapts a field target to database/sql and JSON interfaces.
type JSONFieldWrapper[T any] struct {
	target *T
}

// NewJSONFieldWrapper wraps target so lorm can scan and write it as JSON.
func NewJSONFieldWrapper[T any](target *T) *JSONFieldWrapper[T] {
	return &JSONFieldWrapper[T]{target: target}
}

// Value implements driver.Valuer.
func (s *JSONFieldWrapper[T]) Value() (driver.Value, error) {
	if s.target == nil {
		return nil, nil
	}
	if v, ok := any(s.target).(driver.Valuer); ok {
		return v.Value()
	}
	return json.Marshal(s.target)
}

// String returns the JSON encoding of the wrapped value.
func (s *JSONFieldWrapper[T]) String() string {
	if s.target == nil {
		return ""
	}
	str, _ := json.MarshalString(s.target)
	return str
}

// MarshalJSON returns the wrapped value encoded as JSON.
func (s *JSONFieldWrapper[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.target)
}

// UnmarshalJSON decodes JSON into the wrapped value.
func (s *JSONFieldWrapper[T]) UnmarshalJSON(data []byte) error {
	if s.target == nil {
		return errors.New("lorm: JSON scan target is nil")
	}
	return json.Unmarshal(data, s.target)
}

// Scan implements sql.Scanner for JSON-encoded columns.
func (s *JSONFieldWrapper[T]) Scan(src any) error {
	isNull := src == nil
	switch v := src.(type) {
	case []byte:
		isNull = len(v) == 0 || string(v) == "null"
	case string:
		isNull = v == "" || v == "null"
	}
	if s.target == nil {
		if isNull {
			return nil
		}
		return errors.New("lorm: JSON scan target is nil")
	}
	if v, ok := any(s.target).(sql.Scanner); ok {
		return v.Scan(src)
	}
	if isNull {
		var zero T
		*s.target = zero
		return nil
	}
	// Fall back to JSON decoding for drivers that return raw strings or bytes.
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s.target)
	case string:
		return json.Unmarshal([]byte(v), s.target)
	}
	return fmt.Errorf("cannot unmarshal %v into %T", src, s.target)
}
