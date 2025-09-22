package builder

import (
	"database/sql"
	"fmt"
	"sync"
)

// StmtCache wraps and delegates down to a Preparer type
//
// It also automatically prepares all statements sent to the underlying Preparer calls
// for Exec, Query and QueryRow and caches the returns *sql.Stmt using the provided
// query as the key. So that it can be automatically re-used.
type StmtCache struct {
	cache map[string]*sql.Stmt
	mu    sync.Mutex
}

// GetOrStore delegates down to the underlying Preparer and caches the result
// using the provided query as a key
func (sc *StmtCache) GetOrStore(query string, f func() (*sql.Stmt, error)) (*sql.Stmt, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	stmt, ok := sc.cache[query]
	if ok {
		return stmt, nil
	}
	stmt, err := f()
	if err == nil {
		sc.cache[query] = stmt
	}
	return stmt, err
}

// Clear removes and closes all the currently cached prepared statements
func (sc *StmtCache) Clear() (err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for key, stmt := range sc.cache {
		delete(sc.cache, key)

		if stmt == nil {
			continue
		}

		if cerr := stmt.Close(); cerr != nil {
			err = cerr
		}
	}

	if err != nil {
		return fmt.Errorf("one or more Stmt.Close failed; last error: %v", err)
	}

	return
}
