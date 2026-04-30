package ormcrud

import (
	"time"

	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

type GormUser struct {
	ID        int64                  `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string                 `gorm:"column:name;not null;index"`
	Alias     *string                `gorm:"column:alias"`
	Age       int                    `gorm:"column:age;not null"`
	AgeP      *int                   `gorm:"column:age_p"`
	Active    bool                   `gorm:"column:active;not null"`
	ActiveP   *bool                  `gorm:"column:active_p"`
	Email     string                 `gorm:"column:email;not null;uniqueIndex"`
	Tags      benchmodel.IntSlice    `gorm:"column:tags;not null"`
	Meta      benchmodel.StringMap   `gorm:"column:meta;not null"`
	Profile   benchmodel.Profile     `gorm:"column:profile;not null"`
	Contacts  benchmodel.ContactList `gorm:"column:contacts;not null"`
	CreatedAt time.Time              `gorm:"column:created_at;not null"`
	UpdatedAt time.Time              `gorm:"column:updated_at;not null"`
}

func (GormUser) TableName() string { return "bench_users" }
