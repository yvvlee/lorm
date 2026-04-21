package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type Preferences struct {
	Theme         string `json:"theme"`
	EmailsEnabled bool   `json:"emails_enabled"`
}

type UserProfile struct {
	lorm.UnimplementedTable `lorm:"user_profiles"`
	ID                      int64       `lorm:"id,primary_key,auto_increment"`
	Name                    string      `lorm:"name"`
	Tags                    []string    `lorm:"tags,json"`
	Preferences             Preferences `lorm:"preferences,json"`
	CreatedAt               time.Time   `lorm:"created_at,created"`
	UpdatedAt               time.Time   `lorm:"updated_at,updated"`
}
