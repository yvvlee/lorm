package testdata

import (
	"time"

	"github.com/yvvlee/lorm"
)

type User struct {
	lorm.UnimplementedTable `lorm:"users"`
	ID                      int `lorm:"primary_key,auto_increment"`
	Name                    string
	Age                     int
	CreatedAt               time.Time `lorm:"created"`
	UpdatedAt               time.Time `lorm:"updated"`
}
