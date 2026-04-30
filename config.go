package lorm

import (
	"time"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

// IgnoreStrategy defines how INSERT IGNORE is expressed in different SQL dialects.
type IgnoreStrategy uint8

const (
	// IgnoreKeyword uses "INSERT IGNORE INTO ..." (MySQL/MariaDB default).
	IgnoreKeyword IgnoreStrategy = iota
	// IgnoreOrKeyword uses "INSERT OR IGNORE INTO ..." (SQLite).
	IgnoreOrKeyword
	// IgnoreConflictSuffix appends "ON CONFLICT DO NOTHING" (PostgreSQL).
	IgnoreConflictSuffix
)

// DialectConfig holds SQL dialect behavior used by an Engine.
type DialectConfig struct {
	// PlaceholderFormat specifies the format of placeholders used in SQL queries (e.g. ?, $1, :1).
	PlaceholderFormat builder.PlaceholderFormat
	// Escaper escapes table names and column names.
	Escaper names.Escaper
	// SupportsReturning indicates whether the dialect supports RETURNING clauses.
	SupportsReturning bool
	// SupportsLastInsertID indicates whether the dialect supports LastInsertId.
	SupportsLastInsertID bool
	// SupportsForUpdate indicates whether the dialect supports FOR UPDATE.
	SupportsForUpdate bool
	// IgnoreStrategy defines the dialect-specific INSERT IGNORE syntax.
	IgnoreStrategy IgnoreStrategy
}

// Config holds engine settings assembled from Option values.
type Config struct {
	// driverName is the name of the database driver (e.g. "mysql", "postgres", etc.)
	driverName string
	// dsn is the data source name or connection string used to connect to the database
	dsn string
	// Dialect holds SQL dialect behavior for generated statements.
	Dialect DialectConfig
	// logger is the logger instance used for logging database operations
	logger Logger
	// logSQLArgs controls whether SQL argument values are included in engine logs
	logSQLArgs bool
	// maxIdleConns is the maximum number of idle connections in the connection pool
	maxIdleConns int
	// maxOpenConns is the maximum number of open connections to the database
	maxOpenConns int
	// connMaxLifetime is the maximum amount of time a connection may be reused
	connMaxLifetime time.Duration
	// connMaxIdleTime is the maximum amount of time a connection may be idle
	connMaxIdleTime time.Duration
}

// Option mutates Config during engine construction.
type Option func(*Config)

// WithDialectConfig sets the dialect behavior as a single config value.
func WithDialectConfig(dialect DialectConfig) Option {
	return func(c *Config) {
		c.Dialect = dialect
	}
}

// WithPlaceholderFormat sets the placeholder format
func WithPlaceholderFormat(format builder.PlaceholderFormat) Option {
	return func(c *Config) {
		c.Dialect.PlaceholderFormat = format
	}
}

// WithEscaper sets the escaper
func WithEscaper(escaper names.Escaper) Option {
	return func(c *Config) {
		c.Dialect.Escaper = escaper
	}
}

// WithSupportsReturning overrides RETURNING support detection.
func WithSupportsReturning(enabled bool) Option {
	return func(c *Config) {
		c.Dialect.SupportsReturning = enabled
	}
}

// WithSupportsLastInsertID overrides LastInsertId support detection.
func WithSupportsLastInsertID(enabled bool) Option {
	return func(c *Config) {
		c.Dialect.SupportsLastInsertID = enabled
	}
}

// WithSupportsForUpdate overrides FOR UPDATE support detection.
func WithSupportsForUpdate(enabled bool) Option {
	return func(c *Config) {
		c.Dialect.SupportsForUpdate = enabled
	}
}

// WithMaxIdleConns sets the maximum number of idle connections
func WithMaxIdleConns(maxIdleConns int) Option {
	return func(c *Config) {
		c.maxIdleConns = maxIdleConns
	}
}

// WithMaxOpenConns sets the maximum number of open connections
func WithMaxOpenConns(maxOpenConns int) Option {
	return func(c *Config) {
		c.maxOpenConns = maxOpenConns
	}
}

// WithConnMaxLifetime sets the maximum lifetime of a connection
func WithConnMaxLifetime(connMaxLifetime time.Duration) Option {
	return func(c *Config) {
		c.connMaxLifetime = connMaxLifetime
	}
}

// WithConnMaxIdleTime sets the maximum idle time of a connection
func WithConnMaxIdleTime(connMaxIdleTime time.Duration) Option {
	return func(c *Config) {
		c.connMaxIdleTime = connMaxIdleTime
	}
}

// WithLogger sets the logger
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithLogSQLArgs controls whether engine logs include SQL argument values.
func WithLogSQLArgs(enabled bool) Option {
	return func(c *Config) {
		c.logSQLArgs = enabled
	}
}
