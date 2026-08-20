package main

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yvvlee/lorm"
)

type CSVInts []int

var _ lorm.ScannerValuer = (*CSVInts)(nil)

func (c CSVInts) Value() (driver.Value, error) {
	if len(c) == 0 {
		return []byte{}, nil
	}
	parts := make([]string, len(c))
	for i, item := range c {
		parts[i] = strconv.Itoa(item)
	}
	return []byte(strings.Join(parts, ",")), nil
}

func (c *CSVInts) Scan(src any) error {
	if c == nil {
		return fmt.Errorf("CSVInts destination is nil")
	}
	var data []byte
	switch v := src.(type) {
	case nil:
		*c = nil
		return nil
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into CSVInts", src)
	}
	if len(data) == 0 {
		*c = nil
		return nil
	}
	parts := strings.Split(string(data), ",")
	result := make([]int, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return err
		}
		result[i] = value
	}
	*c = result
	return nil
}

type Report struct {
	lorm.UnimplementedTable `lorm:"reports"`
	ID                      int64     `lorm:"id,primary_key,auto_increment"`
	Title                   string    `lorm:"title"`
	Scores                  CSVInts   `lorm:"scores"`
	CreatedAt               time.Time `lorm:"created_at,created"`
	UpdatedAt               time.Time `lorm:"updated_at,updated"`
}
