package benchmodel

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringMap map[string]string

type IntSlice []int

type Profile struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Active bool     `json:"active"`
	Labels []string `json:"labels"`
}

type Contact struct {
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type ContactList []Contact

func (m StringMap) Value() (driver.Value, error) {
	return marshalJSON(m)
}

func (m *StringMap) Scan(src any) error {
	return scanJSON(src, m)
}

func (m StringMap) ToDB() ([]byte, error) {
	return marshalJSONBytes(m)
}

func (m *StringMap) FromDB(src []byte) error {
	return scanJSON(src, m)
}

func (s IntSlice) Value() (driver.Value, error) {
	return marshalJSON(s)
}

func (s *IntSlice) Scan(src any) error {
	return scanJSON(src, s)
}

func (s IntSlice) ToDB() ([]byte, error) {
	return marshalJSONBytes(s)
}

func (s *IntSlice) FromDB(src []byte) error {
	return scanJSON(src, s)
}

func (p Profile) Value() (driver.Value, error) {
	return marshalJSON(p)
}

func (p *Profile) Scan(src any) error {
	return scanJSON(src, p)
}

func (p Profile) ToDB() ([]byte, error) {
	return marshalJSONBytes(p)
}

func (p *Profile) FromDB(src []byte) error {
	return scanJSON(src, p)
}

func (c ContactList) Value() (driver.Value, error) {
	return marshalJSON(c)
}

func (c *ContactList) Scan(src any) error {
	return scanJSON(src, c)
}

func (c ContactList) ToDB() ([]byte, error) {
	return marshalJSONBytes(c)
}

func (c *ContactList) FromDB(src []byte) error {
	return scanJSON(src, c)
}

func marshalJSON(v any) (driver.Value, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func marshalJSONBytes(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func scanJSON(src any, dest any) error {
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, dest)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), dest)
	default:
		return fmt.Errorf("unsupported benchmark JSON source type %T", src)
	}
}
