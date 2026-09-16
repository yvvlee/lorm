package builder

import "strings"

type conditionKind uint8

const (
	conditionEmpty conditionKind = iota
	conditionTrue
	conditionFalse
	conditionExpression
)

// Keep absence separate from truth until the enclosing group or clause is built.
type conditionSQL struct {
	sql  string
	args []any
	kind conditionKind
}

func constantCondition(kind conditionKind) conditionSQL {
	result := conditionSQL{kind: kind, args: []any{}}
	switch kind {
	case conditionTrue:
		result.sql = sqlTrue
	case conditionFalse:
		result.sql = sqlFalse
	}
	return result
}

func buildCondition(pred Sqlizer) (conditionSQL, error) {
	if pred == nil {
		return constantCondition(conditionEmpty), nil
	}
	switch pred := pred.(type) {
	case And:
		condition, err := buildConditions(pred, false, true)
		if err == nil && condition.kind == conditionEmpty {
			condition = constantCondition(conditionTrue)
		}
		return condition, err
	case Or:
		return buildConditions(pred, true, true)
	case *wherePart:
		if nested, ok := pred.pred.(Sqlizer); ok {
			return buildCondition(nested)
		}
	}
	sql, args, err := pred.ToSql()
	if err != nil {
		return conditionSQL{}, err
	}
	return conditionSQL{sql: sql, args: args, kind: classifyCondition(sql, args)}, nil
}

// Only recognize standalone constants. Arbitrary raw SQL remains opaque.
func classifyCondition(sql string, args []any) conditionKind {
	condition := strings.TrimSpace(sql)
	if condition == "" {
		return conditionEmpty
	}
	if len(args) > 0 {
		return conditionExpression
	}
	for strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		condition = strings.TrimSpace(condition[1 : len(condition)-1])
	}
	switch strings.ToUpper(condition) {
	case "TRUE":
		return conditionTrue
	case "FALSE":
		return conditionFalse
	}
	left, right, ok := strings.Cut(condition, "=")
	if ok && strings.TrimSpace(left) == "1" {
		switch strings.TrimSpace(right) {
		case "1":
			return conditionTrue
		case "0":
			return conditionFalse
		}
	}
	return conditionExpression
}

func buildConditions(parts []Sqlizer, or, wrap bool) (conditionSQL, error) {
	var sqlParts []string
	var args []any
	var hasTrue, hasFalse bool
	for _, part := range parts {
		condition, err := buildCondition(part)
		if err != nil {
			return conditionSQL{}, err
		}
		// Visit every child so a constant cannot hide a later build error.
		switch condition.kind {
		case conditionTrue:
			hasTrue = true
		case conditionFalse:
			hasFalse = true
		case conditionExpression:
			sqlParts = append(sqlParts, condition.sql)
			args = append(args, condition.args...)
		}
	}
	if or && hasTrue {
		return constantCondition(conditionTrue), nil
	}
	if !or && hasFalse {
		return constantCondition(conditionFalse), nil
	}
	if len(sqlParts) == 0 {
		if !or && hasTrue {
			return constantCondition(conditionTrue), nil
		}
		if hasFalse {
			return constantCondition(conditionFalse), nil
		}
		return constantCondition(conditionEmpty), nil
	}
	separator := " AND "
	if or {
		separator = " OR "
	}
	sql := strings.Join(sqlParts, separator)
	if wrap {
		sql = "(" + sql + ")"
	}
	return conditionSQL{sql: sql, args: args, kind: conditionExpression}, nil
}
