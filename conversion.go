package lorm

import (
	"database/sql/driver"
	"reflect"
)

// adaptDBArgs prepares query arguments for database/sql by resolving
// pointers and driver.Valuer implementations.
func adaptDBArgs(args []any) []any {
	if len(args) == 0 {
		return args
	}
	adapted := make([]any, len(args))
	for i, arg := range args {
		adapted[i] = normalizeDBArg(arg)
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
		if _, ok := current.(driver.Valuer); ok {
			return current
		}
		value = value.Elem()
	}
	return nil
}
