package testdata

import (
	"github.com/yvvlee/lorm"
)

type Address struct {
	Address  string
	PostCode string
}

type UserAddress struct {
	lorm.UnimplementedModel
	ID         int        `lorm:"primary_key,auto_increment"`
	Int64Alias Int64Alias `lorm:"int64_alias"`
	Strings    Strings    `lorm:"strings,json"`
	Address    `lorm:"addr_"`
}

type Int64Alias int64

type Strings = []string
