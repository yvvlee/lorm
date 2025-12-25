package try

import (
	"cmp"
	"strings"
	"time"

	"github.com/yvvlee/lorm/builder"
)

type ordered interface {
	cmp.Ordered | ~bool
}

// Equal adds condition dbField = value if value is not nil
func Equal[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Eq{dbField: *value}
}

// NotEqual adds condition dbField != value if value is not nil
func NotEqual[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.NotEq{dbField: *value}
}

// Gt adds condition dbField > value if value is not nil
func Gt[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Gt{dbField: *value}
}

// Gte adds condition dbField >= value if value is not nil
func Gte[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Gte{dbField: *value}
}

// Lt adds condition dbField < value if value is not nil
func Lt[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Lt{dbField: *value}
}

// Lte adds condition dbField <= value if value is not nil
func Lte[T ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Lte{dbField: *value}
}

// Like adds condition dbField like "%${value}%" if value is not empty
func Like(dbField, value string) builder.Sqlizer {
	if v := strings.TrimSpace(value); v != "" {
		return builder.Like{dbField, v}
	}
	return nil
}

// Likes adds conditions dbField like "%${value1}%" OR dbField like "%${value2}%" if values are not empty
func Likes(dbField string, values []string) builder.Sqlizer {
	if len(values) == 0 {
		return nil
	}
	var c []builder.Sqlizer
	for _, v := range values {
		c = append(c, builder.Like{dbField, v})
	}
	return builder.Or(c)
}

// Range adds condition dbField >= min if min is not nil, and dbField <= max if max is not nil
func Range[T ordered](dbField string, min, max *T) builder.Sqlizer {
	if min == nil {
		if max == nil {
			return nil
		} else {
			return builder.Lte{dbField: *max}
		}
	} else {
		if max == nil {
			return builder.Gte{dbField: *min}
		} else {
			return builder.And{
				builder.Gte{dbField: *min},
				builder.Lte{dbField: *max},
			}
		}
	}
}

func timeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateTime)
}

// TimeRange adds condition dbField >= start if start is not zero, and dbField < end if end is not zero
func TimeRange(dbField string, start, end *time.Time) builder.Sqlizer {
	if start == nil || start.IsZero() {
		if end == nil || end.IsZero() {
			return nil
		} else {
			return builder.Lt{dbField: timeToString(end)}
		}
	} else {
		if end == nil || end.IsZero() {
			return builder.Gte{dbField: timeToString(start)}
		} else {
			return builder.And{
				builder.Gte{dbField: timeToString(start)},
				builder.Lt{dbField: timeToString(end)},
			}
		}
	}
}

// MultiLike adds conditions dbField1 like "%${value}%" OR dbField2 like "%${value}%" if value is not empty
func MultiLike(dbFields []string, value string) builder.Sqlizer {
	if v := strings.TrimSpace(value); v != "" {
		var conds []builder.Sqlizer
		for _, field := range dbFields {
			conds = append(conds, builder.Like{field, v})
		}
		return builder.Or(conds)
	}
	return nil
}

// In adds condition dbField IN (values) if values are not empty
func In[T any](dbField string, values *[]T) builder.Sqlizer {
	if values == nil || len(*values) == 0 {
		return nil
	}
	return builder.In{Col: dbField, Val: *values}
}

// NotIn adds condition dbField NOT IN (values) if values are not empty
func NotIn[T any](dbField string, values *[]T) builder.Sqlizer {
	if values == nil || len(*values) == 0 {
		return nil
	}
	return builder.NotIn{Col: dbField, Val: *values}
}
