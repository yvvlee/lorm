package ormcrud

import (
	"context"
	"time"

	"github.com/yvvlee/lorm"
)

type noopLogger struct{}

func (noopLogger) DebugContext(context.Context, string, ...any) {}
func (noopLogger) InfoContext(context.Context, string, ...any)  {}
func (noopLogger) WarnContext(context.Context, string, ...any)  {}
func (noopLogger) ErrorContext(context.Context, string, ...any) {}

type LormUser struct {
	lorm.UnimplementedTable
	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Age       int       `lorm:"age"`
	Email     string    `lorm:"email"`
	CreatedAt time.Time `lorm:"created_at,created"`
	UpdatedAt time.Time `lorm:"updated_at,updated"`
}

func (*LormUser) TableName() string { return "bench_users" }

func (*LormUser) New() lorm.Model { return new(LormUser) }

func (u *LormUser) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &u.ID
	case "name":
		return &u.Name
	case "age":
		return &u.Age
	case "email":
		return &u.Email
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
		{Name: "Age", FullName: "Age", DBField: "age"},
		{Name: "Email", FullName: "Email", DBField: "email"},
		{Name: "CreatedAt", FullName: "CreatedAt", DBField: "created_at", Flag: lorm.FlagCreated},
		{Name: "UpdatedAt", FullName: "UpdatedAt", DBField: "updated_at", Flag: lorm.FlagUpdated},
	},
}
