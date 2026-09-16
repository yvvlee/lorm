package lorm

import "github.com/yvvlee/lorm/builder"

func statementSQL(engine *Engine, stmt builder.Sqlizer) (string, []any, error) {
	query, args, err := stmt.ToSql()
	if err != nil {
		return "", nil, err
	}
	// Match session.Exec/Query: queries without arguments are passed through.
	if len(args) > 0 {
		query, err = engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return "", nil, err
		}
	}
	return query, args, nil
}
