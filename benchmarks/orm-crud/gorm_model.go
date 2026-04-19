package ormcrud

import "time"

type GormUser struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name;not null;index"`
	Age       int       `gorm:"column:age;not null"`
	Email     string    `gorm:"column:email;not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (GormUser) TableName() string { return "bench_users" }
