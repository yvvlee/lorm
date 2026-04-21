package main

import "github.com/yvvlee/lorm"

type Account struct {
	lorm.UnimplementedTable `lorm:"accounts"`
	ID                      int64 `lorm:"id,primary_key,auto_increment"`
	Owner                   string
	Balance                 int64
}
