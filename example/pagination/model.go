package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type Post struct {
	lorm.UnimplementedTable `lorm:"posts"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Title                   string    `lorm:"title"`
	Category                string    `lorm:"category"`
	CreatedAt               time.Time `lorm:"created_at,created"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}
