package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

//go:embed schema.sql
var schemaSQL string

func main() {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	admin := &Role{Name: "admin"}
	if _, err := lorm.Insert[*Role](engine).AddModel(admin).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	viewer := &Role{Name: "viewer"}
	if _, err := lorm.Insert[*Role](engine).AddModel(viewer).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	if _, err := lorm.Insert[*User](engine).AddModels(
		&User{Name: "Alice", Email: "alice@example.com", RoleID: admin.ID},
		&User{Name: "Bob", Email: "bob@example.com", RoleID: viewer.ID},
	).Exec(ctx); err != nil {
		log.Fatal(err)
	}

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
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("users with role names:")
	for _, item := range rows {
		fmt.Printf("- user_id=%d user_name=%s email=%s role_name=%s\n", item.UserID, item.UserName, item.Email, item.RoleName)
	}
}
