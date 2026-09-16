package lorm

import (
	"context"
	"database/sql"
	"errors"
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
	}
	return proxy.QueryContext(ctx, query, args...)
}

func (s *session) Exist(ctx context.Context, query string, args ...any) (exist bool, err error) {
	proxy := s.proxy()
	if len(args) > 0 {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
	}
	rows, err := proxy.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	exist = rows.Next()
	err = rows.Err()
	if closeErr := rows.Close(); closeErr != nil {
		return false, errors.Join(err, closeErr)
	}
	if err != nil {
		return false, err
	}
	return
}

func (s *session) proxy() DBProxy {
	if s.tx != nil {
		return s.tx
	}
	return s.engine.db
}
