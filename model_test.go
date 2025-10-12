package lorm

import (
	"time"

	"github.com/shopspring/decimal"
)

type Test struct {
	UnimplementedTable
	ID         uint64 `lorm:"primary_key,auto_increment"`
	Int        int    `lorm:"index"` // db field is "index", struct field is "Int"
	IntP       *int
	Bool       bool
	BoolP      *bool
	Str        string
	StrP       *string
	Timestamp  time.Time
	TimestampP *time.Time
	Datetime   time.Time
	DatetimeP  *time.Time
	Decimal    decimal.Decimal
	DecimalP   *decimal.Decimal
	IntSlice   []int     `lorm:"json"`
	IntSliceP  *[]int    `lorm:"json"`
	Struct     Sub       `lorm:"json"`
	StructP    *Sub      `lorm:"json"`
	CreatedAt  time.Time `lorm:"created"`
	UpdatedAt  time.Time `lorm:"updated"`
}

type Sub struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
