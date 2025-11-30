package try

import (
	"cmp"
	"strings"
	"time"

	"github.com/yvvlee/lorm/builder"
)

type Ordered interface {
	cmp.Ordered | ~bool
}

// Equal 如果value不为空，则添加 dbField = value 的条件
func Equal[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Eq{dbField: *value}
}

// NotEqual 如果value不为空，则添加 dbField != value 的条件
func NotEqual[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.NotEq{dbField: *value}
}

// Gt 如果value不为空，则添加 dbField > value 的条件
func Gt[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Gt{dbField: *value}
}

// Gte 如果value不为空，则添加 dbField >= value 的条件
func Gte[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Gte{dbField: *value}
}

// Lt 如果value不为空，则添加 dbField < value 的条件
func Lt[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Lt{dbField: *value}
}

// Lte 如果value不为空，则添加 dbField <= value 的条件
func Lte[T Ordered](dbField string, value *T) builder.Sqlizer {
	if value == nil {
		return nil
	}
	return builder.Lte{dbField: *value}
}

// Like 如果value不为空，则添加 dbField like "%${value}%" 的条件
func Like(dbField, value string) builder.Sqlizer {
	if v := strings.TrimSpace(value); v != "" {
		return builder.Like{dbField: v}
	}
	return nil
}

// Likes 如果values不为空，则添加 dbField like "%${value1}%" OR dbField like "%${value2}%" 的条件
func Likes(dbField string, values []string) builder.Sqlizer {
	if len(values) == 0 {
		return nil
	}
	var c []builder.Sqlizer
	for _, v := range values {
		c = append(c, builder.Like{dbField: v})
	}
	return builder.Or(c)
}

// Range 如果min不为空，则添加 dbField >= min 的条件;如果max不为空，则添加 dbField <= max 的条件
func Range[T Ordered](dbField string, min, max *T) builder.Sqlizer {
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
	return t.Format("2006-01-02 15:04:05")
}

// TimeRange 如果start不为空，则添加 dbField >= min 的条件;如果end不为空，则添加 dbField < max 的条件
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

// MultiLike 如果value不为空，则添加 dbField1 like "%${value}%" OR dbField2 like "%${value}%" 的条件
func MultiLike(dbFields []string, value string) builder.Sqlizer {
	if v := strings.TrimSpace(value); v != "" {
		var conds []builder.Sqlizer
		for _, field := range dbFields {
			conds = append(conds, builder.Like{field: v})
		}
		return builder.Or(conds)
	}
	return nil
}

// In 如果values不为空，则添加 dbField IN (values) 的条件
func In[T any](dbField string, values *[]T) builder.Sqlizer {
	if values == nil || len(*values) == 0 {
		return nil
	}
	return builder.In{Col: dbField, Val: *values}
}

// NotIn 如果values不为空，则添加 dbField NOT IN (values) 的条件
func NotIn[T any](dbField string, values *[]T) builder.Sqlizer {
	if values == nil || len(*values) == 0 {
		return nil
	}
	return builder.NotIn{Col: dbField, Val: *values}
}
