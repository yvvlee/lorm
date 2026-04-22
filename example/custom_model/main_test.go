package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestCustomModelFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	admin := &Role{Name: "admin"}
	_, err = lorm.Insert[*Role](engine).AddModel(admin).Exec(ctx)
	assert.NoError(t, err)

	viewer := &Role{Name: "viewer"}
	_, err = lorm.Insert[*Role](engine).AddModel(viewer).Exec(ctx)
	assert.NoError(t, err)

	_, err = lorm.Insert[*User](engine).AddModels(
		&User{Name: "Alice", Email: "alice@example.com", RoleID: admin.ID},
		&User{Name: "Bob", Email: "bob@example.com", RoleID: viewer.ID},
	).Exec(ctx)
	assert.NoError(t, err)

	var (
		user User
		role Role
		u    = user.Fields().WithAlias("u")
		r    = role.Fields().WithAlias("r")
	)

	rows, err := lorm.Query[*UserWithRole](engine).
		Select(
			u.ID()+" AS user_id",
			u.Name()+" AS user_name",
			u.Email()+" AS email",
			r.Name()+" AS role_name",
		).
		From(user.TableName() + " AS u").
		InnerJoin(role.TableName() + " AS r ON " + u.RoleID() + " = " + r.ID()).
		OrderBy(u.ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, "admin", rows[0].RoleName)
	assert.Equal(t, "viewer", rows[1].RoleName)
}
