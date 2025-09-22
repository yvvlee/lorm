package builder

import (
	"bytes"
	"fmt"
	"strings"
)

type CountBuilder struct {
	SelectBuilder
}

func (b *CountBuilder) ToSql() (sqlStr string, args []any, err error) {
	if len(b.columns) == 0 {
		err = fmt.Errorf("select statements must have at least one result column")
		return
	}

	sql := &bytes.Buffer{}

	if len(b.prefixes) > 0 {
		args, err = appendToSql(b.prefixes, sql, " ", args)
		if err != nil {
			return
		}

		sql.WriteString(" ")
	}

	sql.WriteString("SELECT ")

	if len(b.options) > 0 {
		sql.WriteString(strings.Join(b.options, " "))
		sql.WriteString(" ")
	}

	if len(b.columns) > 0 {
		args, err = appendToSql(b.columns, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if b.from != nil {
		sql.WriteString(" FROM ")
		args, err = appendToSql([]Sqlizer{b.from}, sql, "", args)
		if err != nil {
			return
		}
	}

	if len(b.joins) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(b.joins, sql, " ", args)
		if err != nil {
			return
		}
	}

	if len(b.whereParts) > 0 {
		sql.WriteString(" WHERE ")
		args, err = appendToSql(b.whereParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(b.groupBys) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(b.groupBys, ", "))
	}

	if len(b.havingParts) > 0 {
		sql.WriteString(" HAVING ")
		args, err = appendToSql(b.havingParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(b.orderByParts) > 0 {
		sql.WriteString(" ORDER BY ")
		args, err = appendToSql(b.orderByParts, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if len(b.suffixes) > 0 {
		sql.WriteString(" ")

		args, err = appendToSql(b.suffixes, sql, " ", args)
		if err != nil {
			return
		}
	}

	sqlStr = sql.String()
	return
}
