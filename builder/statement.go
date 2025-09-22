package builder

// Select returns a new SelectBuilder, optionally setting some result columns.
//
// See SelectBuilder.Select.
func Select(columns ...string) *SelectBuilder {
	return new(SelectBuilder).Select(columns...)
}

// Insert returns a new InsertBuilder with the given table name.
//
// See InsertBuilder.Into.
func Insert(into string) *InsertBuilder {
	return new(InsertBuilder).Into(into)
}

// Replace returns a new InsertBuilder with the statement keyword set to
// "REPLACE" and with the given table name.
//
// See InsertBuilder.Into.
func Replace(into string) *InsertBuilder {
	return Insert(into).StatementKeyword("REPLACE")
}

// Update returns a new UpdateBuilder with the given table name.
//
// See UpdateBuilder.Table.
func Update(table string) *UpdateBuilder {
	return new(UpdateBuilder).Table(table)
}

// Delete returns a new DeleteBuilder with the given table name.
//
// See DeleteBuilder.table.
func Delete(from string) *DeleteBuilder {
	return new(DeleteBuilder).From(from)
}

// Case returns a new CaseBuilder
// "what" represents case value
func Case(what ...any) *CaseBuilder {
	b := &CaseBuilder{}

	switch len(what) {
	case 0:
	case 1:
		b = b.What(what[0])
	default:
		b = b.What(newPart(what[0], what[1:]...))

	}
	return b
}
