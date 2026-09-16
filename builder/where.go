package builder

import (
	"fmt"
	"io"
)

type wherePart part

func newWherePart(pred any, args ...any) Sqlizer {
	return &wherePart{pred: pred, args: args}
}

func (p wherePart) ToSql() (sql string, args []any, err error) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case Sqlizer:
		return pred.ToSql()
	case map[string]any:
		return Eq(pred).ToSql()
	case string:
		sql = pred
		args = p.args
	default:
		err = fmt.Errorf("expected string-keyed map or string, not %T", pred)
	}
	return
}

func hasEffectiveWhere(parts []Sqlizer) bool {
	condition, err := buildConditions(parts, false, false)
	return err == nil && (condition.kind == conditionExpression || condition.kind == conditionFalse)
}

func appendConditionClause(parts []Sqlizer, w io.Writer, clause string, omitTrue bool, args []any) ([]any, error) {
	condition, err := buildConditions(parts, false, false)
	if err != nil {
		return nil, err
	}
	if condition.kind == conditionEmpty || (omitTrue && condition.kind == conditionTrue) {
		return args, nil
	}
	if _, err := io.WriteString(w, clause+condition.sql); err != nil {
		return nil, err
	}
	return append(args, condition.args...), nil
}
