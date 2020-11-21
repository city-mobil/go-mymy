package mymy

type RowsEvent struct {
	Action Action
	Source SourceInfo
	// Rows is a changed row list.
	//
	// Update events has even rows number.
	// Two rows for one update event: [before update row, after update row].
	Rows [][]interface{}
}
