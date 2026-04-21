package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type User struct {
	lorm.UnimplementedTable `lorm:"users"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Name                    string    `lorm:"name"`
	Email                   string    `lorm:"email"`
	CreatedAt               time.Time `lorm:"created_at,created"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}
