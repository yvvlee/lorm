package ormcrud

import (
	"time"

	"github.com/yvvlee/lorm"
	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

type LormUser struct {
	lorm.UnimplementedTable
	ID        int64                  `lorm:"id,primary_key,auto_increment"`
	Name      string                 `lorm:"name"`
	Alias     *string                `lorm:"alias"`
	Age       int                    `lorm:"age"`
	AgeP      *int                   `lorm:"age_p"`
	Active    bool                   `lorm:"active"`
	ActiveP   *bool                  `lorm:"active_p"`
	Email     string                 `lorm:"email"`
	Tags      benchmodel.IntSlice    `lorm:"tags"`
	Meta      benchmodel.StringMap   `lorm:"meta"`
	Profile   benchmodel.Profile     `lorm:"profile"`
	Contacts  benchmodel.ContactList `lorm:"contacts"`
	CreatedAt time.Time              `lorm:"created_at,created"`
	UpdatedAt time.Time              `lorm:"updated_at,updated"`
}

func (*LormUser) TableName() string { return "bench_users" }

func (*LormUser) New() lorm.Model { return new(LormUser) }

func (u *LormUser) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &u.ID
	case "name":
		return &u.Name
	case "alias":
		return &u.Alias
	case "age":
		return &u.Age
	case "age_p":
		return &u.AgeP
	case "active":
		return &u.Active
	case "active_p":
		return &u.ActiveP
	case "email":
		return &u.Email
	case "tags":
		return &u.Tags
	case "meta":
		return &u.Meta
	case "profile":
		return &u.Profile
	case "contacts":
		return &u.Contacts
	case "created_at":
		return &u.CreatedAt
	case "updated_at":
		return &u.UpdatedAt
	default:
		return nil
	}
}

func (u *LormUser) LormScan(row lorm.RowScanner) error {
	return row.Scan(
		&u.ID,
		&u.Name,
		&u.Alias,
		&u.Age,
		&u.AgeP,
		&u.Active,
		&u.ActiveP,
		&u.Email,
		&u.Tags,
		&u.Meta,
		&u.Profile,
		&u.Contacts,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
}

func (*LormUser) LormModelDescriptor() *lorm.ModelDescriptor {
	return lormUserDescriptor
}

var lormUserDescriptor = &lorm.ModelDescriptor{
	Name:        "LormUser",
	TableName:   "bench_users",
	PrimaryKeys: []string{"id"},
	Fields: []*lorm.FieldDescriptor{
		{Name: "ID", FullName: "ID", DBField: "id", Flag: lorm.FlagPrimaryKey | lorm.FlagAutoIncrement},
		{Name: "Name", FullName: "Name", DBField: "name"},
		{Name: "Alias", FullName: "Alias", DBField: "alias"},
		{Name: "Age", FullName: "Age", DBField: "age"},
		{Name: "AgeP", FullName: "AgeP", DBField: "age_p"},
		{Name: "Active", FullName: "Active", DBField: "active"},
		{Name: "ActiveP", FullName: "ActiveP", DBField: "active_p"},
		{Name: "Email", FullName: "Email", DBField: "email"},
		{Name: "Tags", FullName: "Tags", DBField: "tags"},
		{Name: "Meta", FullName: "Meta", DBField: "meta"},
		{Name: "Profile", FullName: "Profile", DBField: "profile"},
		{Name: "Contacts", FullName: "Contacts", DBField: "contacts"},
		{Name: "CreatedAt", FullName: "CreatedAt", DBField: "created_at", Flag: lorm.FlagCreated},
		{Name: "UpdatedAt", FullName: "UpdatedAt", DBField: "updated_at", Flag: lorm.FlagUpdated},
	},
}

var lormUserInsertColumns = []string{
	"id", "name", "alias", "age", "age_p", "active", "active_p", "email",
	"tags", "meta", "profile", "contacts", "created_at", "updated_at",
}

var lormUserInsertColumnsWithoutAutoIncrement = lormUserInsertColumns[1:]

func (u *LormUser) LormBeforeInsert(now lorm.HookTime) lorm.InsertPlan {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	plan := lorm.InsertPlan{
		AutoIncrementColumn: "id",
		AutoIncrementZero:   u.ID == 0,
	}
	if plan.AutoIncrementZero {
		plan.Columns = lormUserInsertColumnsWithoutAutoIncrement
	} else {
		plan.Columns = lormUserInsertColumns
	}
	plan.Values = make([]any, 0, len(plan.Columns))
	if !plan.AutoIncrementZero {
		plan.Values = append(plan.Values, u.ID)
	}
	plan.Values = append(plan.Values,
		u.Name, u.Alias, u.Age, u.AgeP, u.Active, u.ActiveP, u.Email,
		u.Tags, u.Meta, u.Profile, u.Contacts, u.CreatedAt, u.UpdatedAt,
	)
	return plan
}

func (u *LormUser) LormAfterInsert(result lorm.InsertResult) error {
	if result.HasGeneratedID {
		u.ID = result.GeneratedID
	}
	return nil
}

func (u *LormUser) LormBeforeUpdate(now lorm.HookTime) (lorm.UpdatePlan, error) {
	return lorm.UpdatePlan{
		PrimaryKeyCount: 1,
		Where:           []lorm.ColumnValue{{Column: "id", Value: u.ID}},
		Set: []lorm.ColumnValue{
			{Column: "name", Value: u.Name},
			{Column: "alias", Value: u.Alias},
			{Column: "age", Value: u.Age},
			{Column: "age_p", Value: u.AgeP},
			{Column: "active", Value: u.Active},
			{Column: "active_p", Value: u.ActiveP},
			{Column: "email", Value: u.Email},
			{Column: "tags", Value: u.Tags},
			{Column: "meta", Value: u.Meta},
			{Column: "profile", Value: u.Profile},
			{Column: "contacts", Value: u.Contacts},
			{Column: "updated_at", Value: now},
		},
	}, nil
}

func (u *LormUser) LormAfterUpdate(now lorm.HookTime, rowsAffected int64) {
	if rowsAffected > 0 {
		u.UpdatedAt = now
	}
}
