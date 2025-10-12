package lorm

import (
	"time"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type Config struct {
	// driverName is the name of the database driver (e.g. "mysql", "postgres", etc.)
	driverName string
	// dsn is the data source name or connection string used to connect to the database
	dsn string
	// placeholderFormat specifies the format of placeholders used in SQL queries (e.g. ?, $1, :1)
	placeholderFormat builder.PlaceholderFormat
	// escaper is used to escape special characters in table names and column names
	escaper names.Escaper
	// logger is the logger instance used for logging database operations
	logger Logger
	// maxIdleConns is the maximum number of idle connections in the connection pool
	maxIdleConns int
	// maxOpenConns is the maximum number of open connections to the database
	maxOpenConns int
	// connMaxLifetime is the maximum amount of time a connection may be reused
	connMaxLifetime time.Duration
	// connMaxIdleTime is the maximum amount of time a connection may be idle
	connMaxIdleTime time.Duration
}

type Option func(*Config)

// WithPlaceholderFormat sets the placeholder format
func WithPlaceholderFormat(format builder.PlaceholderFormat) Option {
	return func(c *Config) {
		c.placeholderFormat = format
	}
}

// WithEscaper sets the escaper
func WithEscaper(escaper names.Escaper) Option {
	return func(c *Config) {
		c.escaper = escaper
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
