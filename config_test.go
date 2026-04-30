package lorm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type testLogger struct{}

func (testLogger) DebugContext(ctx context.Context, msg string, args ...any) {}
func (testLogger) InfoContext(ctx context.Context, msg string, args ...any)  {}
func (testLogger) WarnContext(ctx context.Context, msg string, args ...any)  {}
func (testLogger) ErrorContext(ctx context.Context, msg string, args ...any) {}

func TestConfigOptions(t *testing.T) {
	c := &Config{}
	WithPlaceholderFormat(builder.Dollar)(c)
	WithEscaper(names.NewQuoter('"', '"'))(c)
	WithMaxIdleConns(3)(c)
	WithMaxOpenConns(9)(c)
	WithConnMaxLifetime(30 * time.Second)(c)
	WithConnMaxIdleTime(10 * time.Second)(c)
	WithLogger(testLogger{})(c)
	WithLogSQLArgs(true)(c)
	WithSupportsReturning(true)(c)
	WithSupportsLastInsertID(false)(c)
	WithSupportsForUpdate(true)(c)

	assert.Equal(t, builder.Dollar, c.Dialect.PlaceholderFormat)
	assert.NotNil(t, c.Dialect.Escaper)
	assert.Equal(t, 3, c.maxIdleConns)
	assert.Equal(t, 9, c.maxOpenConns)
	assert.Equal(t, 30*time.Second, c.connMaxLifetime)
	assert.Equal(t, 10*time.Second, c.connMaxIdleTime)
	assert.True(t, c.logSQLArgs)
	assert.True(t, c.Dialect.SupportsReturning)
	assert.False(t, c.Dialect.SupportsLastInsertID)
	assert.True(t, c.Dialect.SupportsForUpdate)
	_, ok := c.logger.(testLogger)
	assert.True(t, ok)
}

func TestWithDialectConfig(t *testing.T) {
	dialect := DialectConfig{
		PlaceholderFormat:    builder.Dollar,
		Escaper:              names.NewQuoter('"', '"'),
		SupportsReturning:    true,
		SupportsLastInsertID: false,
		SupportsForUpdate:    true,
		IgnoreStrategy:       IgnoreConflictSuffix,
	}
	c := &Config{}

	WithDialectConfig(dialect)(c)

	assert.Equal(t, dialect, c.Dialect)
}
