package testdata

import "github.com/yvvlee/lorm"

type AuditUser struct {
	lorm.UnimplementedModel
	ID int `lorm:"primary_key,auto_increment"`
	AuditFields
}
