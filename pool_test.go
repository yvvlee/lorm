package lorm

import (
	"testing"

	"github.com/yvvlee/lorm/builder"
)

func TestDeleteBuilderPool(t *testing.T) {
	b1 := deleteBuilderPool.Get().(*builder.DeleteBuilder)

	// Simulate usage
	b1.From("users").Where("id = ?", 1)
	sql1, _, _ := b1.ToSql()
	if sql1 == "" {
		t.Error("Expected non-empty SQL before Clear")
	}

	// Clear and return to pool
	b1.Clear()
	deleteBuilderPool.Put(b1)

	// Get from pool
	b2 := deleteBuilderPool.Get().(*builder.DeleteBuilder)

	// Verify builder is clean - generating SQL should fail
	_, _, err := b2.ToSql()
	if err == nil {
		t.Error("Expected error when generating SQL from empty builder")
	}
}

func TestSelectBuilderPool(t *testing.T) {
	b1 := selectBuilderPool.Get().(*builder.SelectBuilder)

	// Simulate usage
	b1.Select("id", "name").From("users")
	sql1, _, _ := b1.ToSql()
	if sql1 == "" {
		t.Error("Expected non-empty SQL before Clear")
	}

	// Clear and return to pool
	b1.Clear()
	selectBuilderPool.Put(b1)

	// Get from pool
	b2 := selectBuilderPool.Get().(*builder.SelectBuilder)

	// Verify builder is clean - generating SQL should fail
	_, _, err := b2.ToSql()
	if err == nil {
		t.Error("Expected error when generating SQL from empty builder")
	}
}

func TestInsertBuilderPool(t *testing.T) {
	b1 := insertBuilderPool.Get().(*builder.InsertBuilder)

	// Simulate usage
	b1.Into("users").Columns("name").Values("test")
	sql1, _, _ := b1.ToSql()
	if sql1 == "" {
		t.Error("Expected non-empty SQL before Clear")
	}

	// Clear and return to pool
	b1.Clear()
	insertBuilderPool.Put(b1)

	// Get from pool
	b2 := insertBuilderPool.Get().(*builder.InsertBuilder)

	// Verify builder is clean - generating SQL should fail
	_, _, err := b2.ToSql()
	if err == nil {
		t.Error("Expected error when generating SQL from empty builder")
	}
}

func TestUpdateBuilderPool(t *testing.T) {
	b1 := updateBuilderPool.Get().(*builder.UpdateBuilder)

	// Simulate usage
	b1.Table("users").Set("name", "test")
	sql1, _, _ := b1.ToSql()
	if sql1 == "" {
		t.Error("Expected non-empty SQL before Clear")
	}

	// Clear and return to pool
	b1.Clear()
	updateBuilderPool.Put(b1)

	// Get from pool
	b2 := updateBuilderPool.Get().(*builder.UpdateBuilder)

	// Verify builder is clean - generating SQL should fail
	_, _, err := b2.ToSql()
	if err == nil {
		t.Error("Expected error when generating SQL from empty builder")
	}
}
