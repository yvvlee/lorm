package builder

import "testing"

func TestInsertBuilderClear(t *testing.T) {
	b := Insert("users").
		Columns("name", "age").
		Values("Alice", 30).
		Prefix("WITH cte AS (SELECT 1)").
		Suffix("RETURNING id")

	b.Clear()

	if b.into != "" {
		t.Errorf("Expected into to be empty, got %s", b.into)
	}
	if len(b.columns) != 0 {
		t.Errorf("Expected columns to be empty, got %v", b.columns)
	}
	if len(b.values) != 0 {
		t.Errorf("Expected values to be empty, got %v", b.values)
	}
	if len(b.prefixes) != 0 {
		t.Errorf("Expected prefixes to be empty, got %v", b.prefixes)
	}
	if len(b.suffixes) != 0 {
		t.Errorf("Expected suffixes to be empty, got %v", b.suffixes)
	}
	if len(b.returning) != 0 {
		t.Errorf("Expected returning to be empty, got %v", b.returning)
	}
}

func TestUpdateBuilderClear(t *testing.T) {
	b := Update("users").
		Set("name", "Bob").
		Where("id = ?", 1).
		OrderBy("id").
		Limit(10)

	b.Clear()

	if b.table != "" {
		t.Errorf("Expected table to be empty, got %s", b.table)
	}
	if len(b.setClauses) != 0 {
		t.Errorf("Expected setClauses to be empty, got %v", b.setClauses)
	}
	if len(b.whereParts) != 0 {
		t.Errorf("Expected whereParts to be empty, got %v", b.whereParts)
	}
	if len(b.orderBys) != 0 {
		t.Errorf("Expected orderBys to be empty, got %v", b.orderBys)
	}
	if b.limit != "" {
		t.Errorf("Expected limit to be empty, got %s", b.limit)
	}
}

func TestSelectBuilderClear(t *testing.T) {
	b := Select("id", "name").
		From("users").
		Where("age > ?", 18).
		GroupBy("name").
		OrderBy("id").
		Limit(10).
		Offset(5)

	b.Clear()

	if len(b.columns) != 0 {
		t.Errorf("Expected columns to be empty, got %v", b.columns)
	}
	if b.from != nil {
		t.Errorf("Expected from to be nil, got %v", b.from)
	}
	if len(b.whereParts) != 0 {
		t.Errorf("Expected whereParts to be empty, got %v", b.whereParts)
	}
	if len(b.groupBys) != 0 {
		t.Errorf("Expected groupBys to be empty, got %v", b.groupBys)
	}
	if len(b.orderByParts) != 0 {
		t.Errorf("Expected orderByParts to be empty, got %v", b.orderByParts)
	}
	if b.limit != "" {
		t.Errorf("Expected limit to be empty, got %s", b.limit)
	}
	if b.offset != "" {
		t.Errorf("Expected offset to be empty, got %s", b.offset)
	}
}

func TestDeleteBuilderClear(t *testing.T) {
	b := Delete("users").
		Where("id = ?", 1).
		OrderBy("id").
		Limit(10)

	b.Clear()

	if b.from != "" {
		t.Errorf("Expected from to be empty, got %s", b.from)
	}
	if len(b.whereParts) != 0 {
		t.Errorf("Expected whereParts to be empty, got %v", b.whereParts)
	}
	if len(b.orderBys) != 0 {
		t.Errorf("Expected orderBys to be empty, got %v", b.orderBys)
	}
	if b.limit != "" {
		t.Errorf("Expected limit to be empty, got %s", b.limit)
	}
}
