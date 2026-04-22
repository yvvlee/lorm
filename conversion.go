package lorm

import (
	"database/sql/driver"
	"fmt"
	"reflect"
)

// ConversionFrom lets a custom field decode a raw database payload into itself.
type ConversionFrom interface {
	FromDB([]byte) error
}

// ConversionTo lets a custom field encode itself into a database payload.
type ConversionTo interface {
	ToDB() ([]byte, error)
}

// Conversion combines the read and write sides of custom field conversion.
type Conversion interface {
	ConversionFrom
	ConversionTo
}

type conversionValuer struct {
	value ConversionTo
}

func (v conversionValuer) Value() (driver.Value, error) {
	return v.value.ToDB()
}

type conversionScanner struct {
	target ConversionFrom
}

func (s conversionScanner) Scan(src any) error {
	data, err := conversionBytes(src)
	if err != nil {
		return err
	}
	return s.target.FromDB(data)
}

func adaptDBArgs(args []any) []any {
	if len(args) == 0 {
		return args
	}
	adapted := make([]any, len(args))
	for i, arg := range args {
		arg = normalizeDBArg(arg)
		if value, ok := asConversionTo(arg); ok {
			adapted[i] = conversionValuer{value: value}
			continue
		}
		adapted[i] = arg
	}
	return adapted
}

func normalizeDBArg(arg any) any {
	if arg == nil {
		return nil
	}
	value := reflect.ValueOf(arg)
	for value.IsValid() {
		if value.Kind() != reflect.Pointer {
			return value.Interface()
		}
		if value.IsNil() {
			return nil
		}
		current := value.Interface()
		if _, ok := current.(ConversionTo); ok {
			return current
		}
		if _, ok := current.(driver.Valuer); ok {
			return current
		}
		value = value.Elem()
	}
	return nil
}

func wrapScanTarget(target any) any {
	if target == nil {
		return target
	}
	if _, ok := target.(interface{ Scan(any) error }); ok {
		return target
	}
	if conversion, ok := asConversionFrom(target); ok {
		return conversionScanner{target: conversion}
	}
	return target
}

func asConversionTo(value any) (ConversionTo, bool) {
	if value == nil {
		return nil, false
	}
	if conversion, ok := value.(ConversionTo); ok {
		return conversion, true
	}
	refValue := reflect.ValueOf(value)
	if refValue.Kind() == reflect.Pointer || !refValue.IsValid() || !refValue.CanAddr() {
		return nil, false
	}
	conversion, ok := refValue.Addr().Interface().(ConversionTo)
	return conversion, ok
}

func asConversionFrom(value any) (ConversionFrom, bool) {
	if value == nil {
		return nil, false
	}
	conversion, ok := value.(ConversionFrom)
	return conversion, ok
}

func conversionBytes(src any) ([]byte, error) {
	switch value := src.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("cannot convert %T into conversion payload", src)
	}
}
