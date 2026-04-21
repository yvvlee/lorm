package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type Document struct {
	lorm.UnimplementedTable `lorm:"documents"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Title                   string    `lorm:"title"`
	Version                 int       `lorm:"version,version"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}
