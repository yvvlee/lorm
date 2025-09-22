package lorm

import (
	"time"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type Config struct {
	driverName        string
	dsn               string
	placeholderFormat builder.PlaceholderFormat
	escaper           names.Escaper
	maxIdleConns      int
	maxOpenConns      int
	connMaxLifetime   time.Duration
	connMaxIdleTime   time.Duration
}

// Option 是一个函数类型，用于配置Config
type Option func(*Config)

// WithPlaceholderFormat 设置占位符格式
func WithPlaceholderFormat(format builder.PlaceholderFormat) Option {
	return func(c *Config) {
		c.placeholderFormat = format
	}
}

// WithEscaper 设置转义器
func WithEscaper(escaper names.Escaper) Option {
	return func(c *Config) {
		c.escaper = escaper
	}
}

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(maxIdleConns int) Option {
	return func(c *Config) {
		c.maxIdleConns = maxIdleConns
	}
}

// WithMaxOpenConns 设置最大打开连接数
func WithMaxOpenConns(maxOpenConns int) Option {
	return func(c *Config) {
		c.maxOpenConns = maxOpenConns
	}
}

// WithConnMaxLifetime 设置连接最大生命周期
func WithConnMaxLifetime(connMaxLifetime time.Duration) Option {
	return func(c *Config) {
		c.connMaxLifetime = connMaxLifetime
	}
}

// WithConnMaxIdleTime 设置连接最大空闲时间
func WithConnMaxIdleTime(connMaxIdleTime time.Duration) Option {
	return func(c *Config) {
		c.connMaxIdleTime = connMaxIdleTime
	}
}
