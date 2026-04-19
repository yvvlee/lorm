package builder

import "testing"

func TestInsertBuilderClear(t *testing.T) {
	b := Insert("users").
		Columns("name", "age").
		Values("Alice", 30).
		Prefix("WITH cte AS (SELECT 1)").
		Suffix("RETURNING id")

	prefixCap := cap(b.prefixes)
	columnsCap := cap(b.columns)
	valuesCap := cap(b.values)
	suffixCap := cap(b.suffixes)
	returningCap := cap(b.returning)

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
	if cap(b.prefixes) != prefixCap || cap(b.columns) != columnsCap || cap(b.values) != valuesCap || cap(b.suffixes) != suffixCap || cap(b.returning) != returningCap {
		t.Fatalf("Expected insert builder slice capacities to be preserved")
	}
	assertStringSliceCleared(t, "columns", b.columns, columnsCap)
	assert2DSliceCleared(t, "values", b.values, valuesCap)
	assertStringSliceCleared(t, "returning", b.returning, returningCap)
}

func TestUpdateBuilderClear(t *testing.T) {
	b := Update("users").
		Set("name", "Bob").
		Where("id = ?", 1).
		OrderBy("id").
		Limit(10)

	setClausesCap := cap(b.setClauses)
	wherePartsCap := cap(b.whereParts)
	orderBysCap := cap(b.orderBys)

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
	if cap(b.setClauses) != setClausesCap || cap(b.whereParts) != wherePartsCap || cap(b.orderBys) != orderBysCap {
		t.Fatalf("Expected update builder slice capacities to be preserved")
	}
	assertSetClauseSliceCleared(t, b.setClauses, setClausesCap)
	assertStringSliceCleared(t, "orderBys", b.orderBys, orderBysCap)
}

func TestSelectBuilderClear(t *testing.T) {
	b := Select("id", "name").
		From("users").
		Where("age > ?", 18).
		GroupBy("name").
		OrderBy("id").
		Limit(10).
		Offset(5)

	columnsCap := cap(b.columns)
	wherePartsCap := cap(b.whereParts)
	groupBysCap := cap(b.groupBys)
	orderByPartsCap := cap(b.orderByParts)

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
	if cap(b.columns) != columnsCap || cap(b.whereParts) != wherePartsCap || cap(b.groupBys) != groupBysCap || cap(b.orderByParts) != orderByPartsCap {
		t.Fatalf("Expected select builder slice capacities to be preserved")
	}
	assertStringSliceCleared(t, "groupBys", b.groupBys, groupBysCap)
}

func TestDeleteBuilderClear(t *testing.T) {
	b := Delete("users").
		Where("id = ?", 1).
		OrderBy("id").
		Limit(10)

	wherePartsCap := cap(b.whereParts)
	orderBysCap := cap(b.orderBys)

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
	if cap(b.whereParts) != wherePartsCap || cap(b.orderBys) != orderBysCap {
		t.Fatalf("Expected delete builder slice capacities to be preserved")
	}
	assertStringSliceCleared(t, "orderBys", b.orderBys, orderBysCap)
}

func assertStringSliceCleared(t *testing.T, name string, items []string, expectedCap int) {
	t.Helper()
	if expectedCap == 0 {
		return
	}
	items = items[:expectedCap]
	for i, item := range items {
		if item != "" {
			t.Fatalf("Expected %s[%d] to be cleared, got %q", name, i, item)
		}
	}
}

func assert2DSliceCleared(t *testing.T, name string, items [][]any, expectedCap int) {
	t.Helper()
	if expectedCap == 0 {
		return
	}
	items = items[:expectedCap]
	for i, item := range items {
		if item != nil {
			t.Fatalf("Expected %s[%d] to be cleared, got %v", name, i, item)
		}
	}
}

func assertSetClauseSliceCleared(t *testing.T, items []setClause, expectedCap int) {
	t.Helper()
	if expectedCap == 0 {
		return
	}
	items = items[:expectedCap]
	for i, item := range items {
		if item.column != "" || item.value != nil {
			t.Fatalf("Expected setClauses[%d] to be cleared, got %+v", i, item)
		}
	}
}
