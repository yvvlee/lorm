package main

import "github.com/yvvlee/lorm"

type Product struct {
	lorm.UnimplementedTable `lorm:"products"`
	ID                      int64  `lorm:"id,primary_key,auto_increment" json:"id"`
	Name                    string `lorm:"name" json:"name"`
	Category                string `lorm:"category" json:"category"`
	Price                   int64  `lorm:"price" json:"price"`
	Active                  bool   `lorm:"active" json:"active"`
}
