package lorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/samber/lo"
)

type Table interface {
	mustEmbedUnimplementedTable()
	TableName
	Model
}

// TableName table name interface to define customerize table name
type TableName interface {
	TableName() string
}

type Model interface {
	mustEmbedUnimplementedModel()
	// New create a new model instance
	New() Model
	// LormFieldMap return field map, key is db field name, value is field value pointer
	LormFieldMap() map[string]any
	// LormModelDescriptor return db fields of this Model
	LormModelDescriptor() *ModelDescriptor
}

type UnimplementedModel struct{}

func (u UnimplementedModel) mustEmbedUnimplementedModel() {}

type UnimplementedTable struct{}

func (u UnimplementedTable) mustEmbedUnimplementedModel() {}
func (u UnimplementedTable) mustEmbedUnimplementedTable() {}

func ModelToInsertData[T Model](model T) (columns []string, values []any) {
	fields, v := ModelsToInsertData([]T{model})
	return fields, v[0]
}

func ModelsToInsertData[T Model](models []T) (columns []string, values [][]any) {
	if len(models) == 0 {
		return
	}
	descriptor := models[0].LormModelDescriptor()
	columns = descriptor.AllFields()
	createdFields := descriptor.FlagFields(FlagCreated)
	updatedFields := descriptor.FlagFields(FlagUpdated)
	jsonFields := descriptor.FlagFields(FlagJson)
	now := time.Now()
	for _, model := range models {
		fieldMap := model.LormFieldMap()
		values = append(values, lo.Map(columns, func(field string, _ int) any {
			if slices.Contains(createdFields, field) || slices.Contains(updatedFields, field) {
				fillCurrentTime(fieldMap[field], now)
			}
			if slices.Contains(jsonFields, field) {
				return NewJSONFieldWrapper(fieldMap[field])
			}
			return fieldMap[field]
		}))
	}
	return
}

type JSONFieldWrapper struct {
	v any
}

func NewJSONFieldWrapper(v any) *JSONFieldWrapper {
	return &JSONFieldWrapper{v: v}
}

func (s *JSONFieldWrapper) Value() (driver.Value, error) {
	if s.v == nil {
		return nil, nil
	}
	return json.Marshal(s.v)
}

func (s *JSONFieldWrapper) String() string {
	if s.v == nil {
		return ""
	}
	str, _ := json.MarshalString(s.v)
	return str
}

func (s *JSONFieldWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.v)
}

func (s *JSONFieldWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, s.v)
}

func (s *JSONFieldWrapper) Scan(src any) error {
	if v, ok := s.v.(sql.Scanner); ok {
		return v.Scan(src)
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s.v)
	case string:
		return json.Unmarshal([]byte(v), s.v)
	case nil:
		return nil
	}
	return fmt.Errorf("cannot unmarshal %v into %t", src, s.v)
}
