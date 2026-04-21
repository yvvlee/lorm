package lorm

import (
	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func escapePredicate(escaper names.Escaper, pred any) any {
	switch v := pred.(type) {
	case map[string]any:
		return escapeMap(escaper, v)
	case builder.Sqlizer:
		return escapeSqlizer(escaper, v)
	default:
		return pred
	}
}

func escapeSqlizer(escaper names.Escaper, sqlizer builder.Sqlizer) builder.Sqlizer {
	switch v := sqlizer.(type) {
	case builder.Eq:
		return builder.Eq(escapeMap(escaper, v))
	case builder.NotEq:
		return builder.NotEq(escapeMap(escaper, builder.Eq(v)))
	case builder.And:
		items := make(builder.And, len(v))
		for i, item := range v {
			items[i] = escapeSqlizer(escaper, item)
		}
		return items
	case builder.Or:
		items := make(builder.Or, len(v))
		for i, item := range v {
			items[i] = escapeSqlizer(escaper, item)
		}
		return items
	default:
		return sqlizer
	}
}

func escapeMap(escaper names.Escaper, m map[string]any) map[string]any {
	if len(m) == 0 {
		return m
	}
	escaped := make(map[string]any, len(m))
	for key, value := range m {
		escaped[escaper.Escape(key)] = value
	}
	return escaped
}
