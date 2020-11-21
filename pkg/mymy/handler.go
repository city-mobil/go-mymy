package mymy

type EventHandler interface {
	OnRows(e *RowsEvent) ([]*Query, error)
}
