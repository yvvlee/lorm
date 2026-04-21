package ormcrud

import (
	"context"
	"time"

	"github.com/yvvlee/lorm"
	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

type noopLogger struct{}

func (noopLogger) DebugContext(context.Context, string, ...any) {}
func (noopLogger) InfoContext(context.Context, string, ...any)  {}
func (noopLogger) WarnContext(context.Context, string, ...any)  {}
func (noopLogger) ErrorContext(context.Context, string, ...any) {}

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

func (*LormUser) LormModelDescriptor() *lorm.ModelDescriptor {
	return lormUserDescriptor
}

var lormUserDescriptor = &lorm.ModelDescriptor{
	Name:      "LormUser",
	TableName: "bench_users",
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
