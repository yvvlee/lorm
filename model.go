package lorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

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
	// LormModelDescriptor return db fields of this Model
	LormModelDescriptor() *ModelDescriptor
}

// UnimplementedModel can be embedded to satisfy the Model marker method.
type UnimplementedModel struct{}

func (u UnimplementedModel) mustEmbedUnimplementedModel() {}

// UnimplementedTable can be embedded to satisfy the Table marker methods.
type UnimplementedTable struct{}

func (u UnimplementedTable) mustEmbedUnimplementedModel() {}
func (u UnimplementedTable) mustEmbedUnimplementedTable() {}

// ModelToInsertData builds insert columns and values for a single model.
func ModelToInsertData[T Model](model T, ignoreFields ...string) (columns []string, values []any) {
	fields, v := ModelsToInsertData([]T{model}, ignoreFields...)
	return fields, v[0]
}

// ModelsToInsertData builds shared insert columns and per-row values for models.
func ModelsToInsertData[T Model](models []T, ignoreFields ...string) (columns []string, values [][]any) {
	if len(models) == 0 {
		return
	}
	descriptor := models[0].LormModelDescriptor()
	ignoreSet := make(map[string]struct{}, len(ignoreFields))
	for _, field := range ignoreFields {
		ignoreSet[field] = struct{}{}
	}

	columns = make([]string, 0, len(descriptor.Fields))
	// Track timestamp-managed columns once so every row can reuse the same now value.
	columnNeedsCurrentTime := make([]bool, 0, len(descriptor.Fields))
	for _, field := range descriptor.Fields {
		if _, ok := ignoreSet[field.DBField]; ok {
			continue
		}
		columns = append(columns, field.DBField)
		columnNeedsCurrentTime = append(columnNeedsCurrentTime, field.Flag.HasFlag(FlagCreated) || field.Flag.HasFlag(FlagUpdated))
	}

	now := time.Now()
	values = make([][]any, 0, len(models))
	for _, model := range models {
		rowValues := make([]any, len(columns))
		for i, field := range columns {
			value := model.LormFieldPtr(field)
			if columnNeedsCurrentTime[i] {
				fillCurrentTime(value, now)
			}
			rowValues[i] = value
		}
		values = append(values, rowValues)
	}
	return
}

// JSONFieldWrapper adapts a field value to database/sql and JSON interfaces.
type JSONFieldWrapper struct {
	v any
}

// NewJSONFieldWrapper wraps v so lorm can scan and write it as JSON.
func NewJSONFieldWrapper(v any) *JSONFieldWrapper {
	return &JSONFieldWrapper{v: v}
}

// Value implements driver.Valuer.
func (s *JSONFieldWrapper) Value() (driver.Value, error) {
	if s.v == nil {
		return nil, nil
	}
	if v, ok := s.v.(driver.Valuer); ok {
		return v.Value()
	}
	return json.Marshal(s.v)
}

// String returns the JSON encoding of the wrapped value.
func (s *JSONFieldWrapper) String() string {
	if s.v == nil {
		return ""
	}
	str, _ := json.MarshalString(s.v)
	return str
}

// MarshalJSON returns the wrapped value encoded as JSON.
func (s *JSONFieldWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.v)
}

// UnmarshalJSON decodes JSON into the wrapped value.
func (s *JSONFieldWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, s.v)
}

// Scan implements sql.Scanner for JSON-encoded columns.
func (s *JSONFieldWrapper) Scan(src any) error {
	if v, ok := s.v.(sql.Scanner); ok {
		return v.Scan(src)
	}
	// Fall back to JSON decoding for drivers that return raw strings or bytes.
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s.v)
	case string:
		return json.Unmarshal([]byte(v), s.v)
	case nil:
		return nil
	}
	return fmt.Errorf("cannot unmarshal %v into %T", src, s.v)
}
