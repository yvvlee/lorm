package test

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yvvlee/lorm"
)

type Test struct {
	lorm.UnimplementedModel
	ID         uint64 `lorm:"primary_key,auto_increment"`
	Int        int    `lorm:"index"`
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
	IntSlice   []int  `lorm:"json"`
	IntSliceP  *[]int `lorm:"json"`
	Struct     Sub    `lorm:"json"`
	StructP    *Sub   `lorm:"json"`
}

type Sub struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
