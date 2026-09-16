package lorm

import (
	"fmt"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func escapePredicate(escaper names.Escaper, pred any) (any, error) {
	switch v := pred.(type) {
	case map[string]any:
		return escapeMap(escaper, v)
	case builder.Sqlizer:
		return escapeSqlizer(escaper, v)
	default:
		return pred, nil
	}
}

func escapeSqlizer(escaper names.Escaper, sqlizer builder.Sqlizer) (builder.Sqlizer, error) {
	switch v := sqlizer.(type) {
	case builder.Eq:
		values, err := escapeMap(escaper, v)
		return builder.Eq(values), err
	case builder.NotEq:
		values, err := escapeMap(escaper, builder.Eq(v))
		return builder.NotEq(values), err
	case builder.FieldExpression:
		return v.WithFieldName(escaper.Escape(v.FieldName())), nil
	case builder.And:
		items := make(builder.And, len(v))
		for i, item := range v {
			var err error
			items[i], err = escapeSqlizer(escaper, item)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	case builder.Or:
		items := make(builder.Or, len(v))
		for i, item := range v {
			var err error
			items[i], err = escapeSqlizer(escaper, item)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	default:
		return sqlizer, nil
	}
}

func escapeMap(escaper names.Escaper, m map[string]any) (map[string]any, error) {
	if len(m) == 0 {
		return m, nil
	}
	escaped := make(map[string]any, len(m))
	for key, value := range m {
		column := escaper.Escape(key)
		if _, exists := escaped[column]; exists {
			return nil, fmt.Errorf("lorm: duplicate column %q after escaping map keys", column)
		}
		escaped[column] = value
	}
	return escaped, nil
}
