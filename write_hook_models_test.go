package lorm

var testInsertColumns = []string{
	"id", "index", "int_p", "bool", "bool_p", "str", "str_p",
	"timestamp", "timestamp_p", "datetime", "datetime_p",
	"decimal", "decimal_p", "int_slice", "int_slice_p",
	"struct", "struct_p", "created_at", "updated_at",
}

var testInsertColumnsWithoutAutoIncrement = testInsertColumns[1:]

func (m *Test) LormBeforeInsert(now HookTime) InsertPlan {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	plan := InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
	}
	if plan.AutoIncrementZero {
		plan.Columns = testInsertColumnsWithoutAutoIncrement
	} else {
		plan.Columns = testInsertColumns
	}
	plan.Values = make([]any, 0, len(plan.Columns))
	if !plan.AutoIncrementZero {
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Values = append(plan.Values,
		m.Int, m.IntP, m.Bool, m.BoolP, m.Str, m.StrP,
		m.Timestamp, m.TimestampP, m.Datetime, m.DatetimeP,
		m.Decimal, m.DecimalP,
		NewJSONFieldWrapper(&m.IntSlice),
		NewJSONFieldWrapper(&m.IntSliceP),
		NewJSONFieldWrapper(&m.Struct),
		NewJSONFieldWrapper(&m.StructP),
		m.CreatedAt, m.UpdatedAt,
	)
	return plan
}

func (m *Test) LormAfterInsert(result InsertResult) error {
	if !result.HasGeneratedID {
		return nil
	}
	value, err := ConvertGeneratedUnsignedID[uint64](result.GeneratedID, 64, "Test.ID")
	if err != nil {
		return err
	}
	m.ID = value
	return nil
}

func (m *Test) LormBeforeUpdate(now HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 1,
		Where: []ColumnValue{
			{Column: "id", Value: m.ID},
		},
		Set: []ColumnValue{
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
			{Column: "int_slice", Value: NewJSONFieldWrapper(&m.IntSlice)},
			{Column: "int_slice_p", Value: NewJSONFieldWrapper(&m.IntSliceP)},
			{Column: "struct", Value: NewJSONFieldWrapper(&m.Struct)},
			{Column: "struct_p", Value: NewJSONFieldWrapper(&m.StructP)},
			{Column: "updated_at", Value: now},
		},
	}, nil
}

func (m *Test) LormAfterUpdate(now HookTime, rowsAffected int64) {
	if rowsAffected > 0 {
		m.UpdatedAt = now
	}
}

func (m *conversionModel) LormBeforeInsert(HookTime) InsertPlan {
	plan := InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
		Columns:             make([]string, 0, 3),
		Values:              make([]any, 0, 3),
	}
	if !plan.AutoIncrementZero {
		plan.Columns = append(plan.Columns, "id")
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Columns = append(plan.Columns, "name", "codes")
	plan.Values = append(plan.Values, m.Name, m.Codes)
	return plan
}

func (m *conversionModel) LormAfterInsert(result InsertResult) error {
	if result.HasGeneratedID {
		m.ID = result.GeneratedID
	}
	return nil
}

func (m *conversionModel) LormBeforeUpdate(HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 1,
		Where:           []ColumnValue{{Column: "id", Value: m.ID}},
		Set: []ColumnValue{
			{Column: "name", Value: m.Name},
			{Column: "codes", Value: m.Codes},
		},
	}, nil
}

func (*conversionModel) LormAfterUpdate(HookTime, int64) {}

func (m *reservedWordModel) LormBeforeInsert(HookTime) InsertPlan {
	plan := InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
		Columns:             make([]string, 0, 2),
		Values:              make([]any, 0, 2),
	}
	if !plan.AutoIncrementZero {
		plan.Columns = append(plan.Columns, "id")
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Columns = append(plan.Columns, "group")
	plan.Values = append(plan.Values, m.Group)
	return plan
}

func (m *reservedWordModel) LormAfterInsert(result InsertResult) error {
	if result.HasGeneratedID {
		m.ID = result.GeneratedID
	}
	return nil
}

func (m *reservedWordModel) LormBeforeUpdate(HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 1,
		Where:           []ColumnValue{{Column: "id", Value: m.ID}},
		Set:             []ColumnValue{{Column: "group", Value: m.Group}},
	}, nil
}

func (*reservedWordModel) LormAfterUpdate(HookTime, int64) {}

func (m *manualPrimaryKeyModel) LormBeforeInsert(HookTime) InsertPlan {
	return InsertPlan{
		Columns: []string{"id", "name"},
		Values:  []any{m.ID, m.Name},
	}
}

func (*manualPrimaryKeyModel) LormAfterInsert(InsertResult) error { return nil }

func (m *manualPrimaryKeyModel) LormBeforeUpdate(HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 1,
		Where:           []ColumnValue{{Column: "id", Value: m.ID}},
		Set:             []ColumnValue{{Column: "name", Value: m.Name}},
	}, nil
}

func (*manualPrimaryKeyModel) LormAfterUpdate(HookTime, int64) {}

func (m *testNoPrimaryKeyModel) LormBeforeInsert(HookTime) InsertPlan {
	return InsertPlan{Columns: []string{"name"}, Values: []any{m.Name}}
}

func (*testNoPrimaryKeyModel) LormAfterInsert(InsertResult) error { return nil }

func (m *testNoPrimaryKeyModel) LormBeforeUpdate(HookTime) (UpdatePlan, error) {
	return UpdatePlan{Set: []ColumnValue{{Column: "name", Value: m.Name}}}, nil
}

func (*testNoPrimaryKeyModel) LormAfterUpdate(HookTime, int64) {}

func (m *testCompositePrimaryKeyModel) LormBeforeInsert(HookTime) InsertPlan {
	return InsertPlan{
		Columns: []string{"account_id", "tenant_id", "name"},
		Values:  []any{m.AccountID, m.TenantID, m.Name},
	}
}

func (*testCompositePrimaryKeyModel) LormAfterInsert(InsertResult) error { return nil }

func (m *testCompositePrimaryKeyModel) LormBeforeUpdate(HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 2,
		Where: []ColumnValue{
			{Column: "account_id", Value: m.AccountID},
			{Column: "tenant_id", Value: m.TenantID},
		},
		Set: []ColumnValue{{Column: "name", Value: m.Name}},
	}, nil
}

func (*testCompositePrimaryKeyModel) LormAfterUpdate(HookTime, int64) {}

func (m *updateSemanticsModel) LormBeforeInsert(now HookTime) InsertPlan {
	plan := InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   m.ID == 0,
		Columns:             make([]string, 0, 4),
		Values:              make([]any, 0, 4),
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if !plan.AutoIncrementZero {
		plan.Columns = append(plan.Columns, "id")
		plan.Values = append(plan.Values, m.ID)
	}
	plan.Columns = append(plan.Columns, "name", "version", "updated_at")
	plan.Values = append(plan.Values, m.Name, m.Version, m.UpdatedAt)
	return plan
}

func (m *updateSemanticsModel) LormAfterInsert(result InsertResult) error {
	if result.HasGeneratedID {
		m.ID = result.GeneratedID
	}
	return nil
}

func (m *updateSemanticsModel) LormBeforeUpdate(now HookTime) (UpdatePlan, error) {
	return UpdatePlan{
		PrimaryKeyCount: 1,
		Where: []ColumnValue{
			{Column: "id", Value: m.ID},
			{Column: "version", Value: m.Version},
		},
		Set: []ColumnValue{
			{Column: "name", Value: m.Name},
			{Column: "updated_at", Value: now},
		},
		Increment: []string{"version"},
	}, nil
}

func (m *updateSemanticsModel) LormAfterUpdate(now HookTime, rowsAffected int64) {
	if rowsAffected > 0 {
		m.UpdatedAt = now
		m.Version++
	}
}
