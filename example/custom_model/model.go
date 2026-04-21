package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type Role struct {
	lorm.UnimplementedTable `lorm:"roles"`
	ID                      int64 `lorm:"id,primary_key,auto_increment"`
	Name                    string
}

type User struct {
	lorm.UnimplementedTable `lorm:"users"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Name                    string    `lorm:"name"`
	Email                   string    `lorm:"email"`
	RoleID                  int64     `lorm:"role_id"`
	CreatedAt               time.Time `lorm:"created_at,created"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}

type UserWithRole struct {
	lorm.UnimplementedModel
	UserID   int64
	UserName string
	Email    string
	RoleName string
}
