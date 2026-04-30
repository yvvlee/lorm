package ormcrud

import (
	"time"

	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

type XormUser struct {
	ID        int64                  `xorm:"'id' pk autoincr"`
	Name      string                 `xorm:"'name' notnull index"`
	Alias     *string                `xorm:"'alias'"`
	Age       int                    `xorm:"'age' notnull"`
	AgeP      *int                   `xorm:"'age_p'"`
	Active    bool                   `xorm:"'active' notnull"`
	ActiveP   *bool                  `xorm:"'active_p'"`
	Email     string                 `xorm:"'email' notnull unique"`
	Tags      benchmodel.IntSlice    `xorm:"'tags' notnull"`
	Meta      benchmodel.StringMap   `xorm:"'meta' notnull"`
	Profile   benchmodel.Profile     `xorm:"'profile' notnull"`
	Contacts  benchmodel.ContactList `xorm:"'contacts' notnull"`
	CreatedAt time.Time              `xorm:"'created_at' created"`
	UpdatedAt time.Time              `xorm:"'updated_at' updated"`
}

func (XormUser) TableName() string { return "bench_users" }
