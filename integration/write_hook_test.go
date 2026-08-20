package integration

import "github.com/yvvlee/lorm"

func (m *Test) LormBeforeInsert(now lorm.HookTime) lorm.InsertPlan {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	plan := lorm.InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
		Columns:             make([]string, 0, 19),
		Values:              make([]any, 0, 19),
	}
	if !plan.AutoIncrementZero {
		plan.Columns = append(plan.Columns, "id")
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Columns = append(plan.Columns,
		"index", "int_p", "bool", "bool_p", "str", "str_p",
		"timestamp", "timestamp_p", "datetime", "datetime_p",
		"decimal", "decimal_p", "int_slice", "int_slice_p",
		"struct", "struct_p", "created_at", "updated_at",
	)
	plan.Values = append(plan.Values,
		m.Int, m.IntP, m.Bool, m.BoolP, m.Str, m.StrP,
		m.Timestamp, m.TimestampP, m.Datetime, m.DatetimeP,
		m.Decimal, m.DecimalP,
		lorm.NewJSONFieldWrapper(&m.IntSlice),
		lorm.NewJSONFieldWrapper(&m.IntSliceP),
		lorm.NewJSONFieldWrapper(&m.Struct),
		lorm.NewJSONFieldWrapper(&m.StructP),
		m.CreatedAt, m.UpdatedAt,
	)
	return plan
}

func (m *Test) LormAfterInsert(result lorm.InsertResult) error {
	if !result.HasGeneratedID {
		return nil
	}
	value, err := lorm.ConvertGeneratedUnsignedID[uint64](result.GeneratedID, 64, "Test.ID")
	if err != nil {
		return err
	}
	m.ID = value
	return nil
}

func (m *Test) LormBeforeUpdate(now lorm.HookTime) (lorm.UpdatePlan, error) {
	return lorm.UpdatePlan{
		PrimaryKeyCount: 1,
		Where: []lorm.ColumnValue{
			{Column: "id", Value: m.ID},
		},
		Set: []lorm.ColumnValue{
			{Column: "index", Value: m.Int},
			{Column: "int_p", Value: m.IntP},
			{Column: "bool", Value: m.Bool},
			{Column: "bool_p", Value: m.BoolP},
			{Column: "str", Value: m.Str},
			{Column: "str_p", Value: m.StrP},
			{Column: "timestamp", Value: m.Timestamp},
			{Column: "timestamp_p", Value: m.TimestampP},
			{Column: "datetime", Value: m.Datetime},
			{Column: "datetime_p", Value: m.DatetimeP},
			{Column: "decimal", Value: m.Decimal},
			{Column: "decimal_p", Value: m.DecimalP},
			{Column: "int_slice", Value: lorm.NewJSONFieldWrapper(&m.IntSlice)},
			{Column: "int_slice_p", Value: lorm.NewJSONFieldWrapper(&m.IntSliceP)},
			{Column: "struct", Value: lorm.NewJSONFieldWrapper(&m.Struct)},
			{Column: "struct_p", Value: lorm.NewJSONFieldWrapper(&m.StructP)},
			{Column: "updated_at", Value: now},
		},
	}, nil
}

func (m *Test) LormAfterUpdate(now lorm.HookTime, rowsAffected int64) {
	if rowsAffected > 0 {
		m.UpdatedAt = now
	}
}

func (m *updateSemanticsModel) LormBeforeInsert(now lorm.HookTime) lorm.InsertPlan {
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	plan := lorm.InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
		Columns:             make([]string, 0, 4),
		Values:              make([]any, 0, 4),
	}
	if !plan.AutoIncrementZero {
		plan.Columns = append(plan.Columns, "id")
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Columns = append(plan.Columns, "name", "version", "updated_at")
	plan.Values = append(plan.Values, m.Name, m.Version, m.UpdatedAt)
	return plan
}

func (m *updateSemanticsModel) LormAfterInsert(result lorm.InsertResult) error {
	if result.HasGeneratedID {
		m.ID = result.GeneratedID
	}
	return nil
}

func (m *updateSemanticsModel) LormBeforeUpdate(now lorm.HookTime) (lorm.UpdatePlan, error) {
	return lorm.UpdatePlan{
		PrimaryKeyCount: 1,
		Where: []lorm.ColumnValue{
			{Column: "id", Value: m.ID},
			{Column: "version", Value: m.Version},
		},
		Set: []lorm.ColumnValue{
			{Column: "name", Value: m.Name},
			{Column: "updated_at", Value: now},
		},
		Increment: []string{"version"},
	}, nil
}

func (m *updateSemanticsModel) LormAfterUpdate(now lorm.HookTime, rowsAffected int64) {
	if rowsAffected > 0 {
		m.UpdatedAt = now
		m.Version++
	}
}
