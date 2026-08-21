package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestCustomModelFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if exampleutil.IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	admin := &Role{Name: "admin"}
	_, err = engine.Insert[*Role]().AddModel(admin).Exec(ctx)
	assert.NoError(t, err)

	viewer := &Role{Name: "viewer"}
	_, err = engine.Insert[*Role]().AddModel(viewer).Exec(ctx)
	assert.NoError(t, err)

	_, err = engine.Insert[*User]().AddModels(
		&User{Name: "Alice", Email: "alice@example.com", RoleID: admin.ID},
		&User{Name: "Bob", Email: "bob@example.com", RoleID: viewer.ID},
	).Exec(ctx)
	assert.NoError(t, err)

	var (
		user User
		role Role
		u    = user.LormCols().WithAlias("u")
		r    = role.LormCols().WithAlias("r")
	)

	rows, err := engine.Query[*UserWithRole]().
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
