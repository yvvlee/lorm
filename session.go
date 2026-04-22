package lorm

import (
	"context"
	"database/sql"
)

type session struct {
	engine *Engine
	tx     *sql.Tx
}

func (s *session) Exec(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	proxy := s.proxy()
	if len(args) > 0 {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
		args = adaptDBArgs(args)
	}
	return proxy.ExecContext(ctx, query, args...)
}

func (s *session) Query(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	proxy := s.proxy()
	if len(args) > 0 {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
		args = adaptDBArgs(args)
	}
	return proxy.QueryContext(ctx, query, args...)
}

func (s *session) Exist(ctx context.Context, query string, args ...any) (exist bool, err error) {
	proxy := s.proxy()
	var rows *sql.Rows
	if len(args) == 0 {
		rows, err = proxy.QueryContext(ctx, query)
	} else {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
		args = adaptDBArgs(args)
		rows, err = proxy.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		exist = true
		return
	}
	err = rows.Err()
	return
}

func (s *session) proxy() DBProxy {
	if s.tx != nil {
		return s.tx
	}
	return s.engine.db
}
