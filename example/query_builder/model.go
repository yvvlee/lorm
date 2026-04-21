package main

import (
	"time"

	"github.com/yvvlee/lorm"
)

type Product struct {
	lorm.UnimplementedTable `lorm:"products"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Name                    string    `lorm:"name"`
	Category                string    `lorm:"category"`
	Price                   int64     `lorm:"price"`
	Status                  string    `lorm:"status"`
	CreatedAt               time.Time `lorm:"created_at,created"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}
