package lorm

import (
	"context"
	"database/sql"
)

type session struct {
	engine   *Engine
	tx       *sql.Tx
	isClosed bool
}

func (s *session) Exec(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	proxy := s.proxy()
	if len(args) == 0 {
		return proxy.ExecContext(ctx, query)
	}
	query, err = s.engine.Placeholder().ReplacePlaceholders(query)
	if err != nil {
		return nil, err
	}
	var stmt *sql.Stmt
	stmt, err = proxy.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	result, err = stmt.ExecContext(ctx, args...)
	return
}

func (s *session) Query(ctx context.Context, scanner Scanner, query string, args ...any) (err error) {
	proxy := s.proxy()
	var rows *sql.Rows
	if len(args) == 0 {
		rows, err = proxy.QueryContext(ctx, query)
	} else {
		query, err = s.engine.Placeholder().ReplacePlaceholders(query)
		if err != nil {
			return
		}
		var stmt *sql.Stmt
		stmt, err = proxy.PrepareContext(ctx, query)
		if err != nil {
			return
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args...)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	err = scanner.Scan(rows)
	return
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
		var stmt *sql.Stmt
		stmt, err = proxy.PrepareContext(ctx, query)
		if err != nil {
			return
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, args...)
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

func (s *session) close() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	if s.tx == nil {
		return nil
	}
	return s.tx.Rollback()
}
func (s *session) commit() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	if s.tx == nil {
		return nil
	}
	return s.tx.Commit()
}
