package testdata

import "time"

type AuditFields struct {
	CreatedAt time.Time  `lorm:"created_at,created"`
	UpdatedAt *time.Time `lorm:"updated_at,updated"`
}
