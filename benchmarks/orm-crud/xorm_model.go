package ormcrud

import "time"

type XormUser struct {
	ID        int64     `xorm:"'id' pk autoincr"`
	Name      string    `xorm:"'name' notnull index"`
	Age       int       `xorm:"'age' notnull"`
	Email     string    `xorm:"'email' notnull unique"`
	CreatedAt time.Time `xorm:"'created_at' created"`
	UpdatedAt time.Time `xorm:"'updated_at' updated"`
}

func (XormUser) TableName() string { return "bench_users" }
