package mymy

import "fmt"

// EventHandler handles incoming events from the master.
type EventHandler interface {
	OnTableChanged(info SourceInfo) error
	OnRows(e *RowsEvent) ([]*Query, error)
}

// BaseEventHandler is a default implementation of the EventHandler.
// Use it as base for your custom handlers in the plugins.
type BaseEventHandler struct {
	table string
	sync  map[string]struct{}
	skip  map[string]struct{}
}

func NewBaseEventHandler(table string) *BaseEventHandler {
	return &BaseEventHandler{
		table: table,
	}
}

// SyncOnly sets list of columns which should be replicated.
//
// This option is mutually exclusive with Skip.
func (eH *BaseEventHandler) SyncOnly(cols []string) {
	sync := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		sync[col] = struct{}{}
	}
	eH.skip = nil
	eH.sync = sync
}

// Skip sets list of columns which should be skipped during the replication.
//
// This option is mutually exclusive with SyncOnly.
func (eH *BaseEventHandler) Skip(cols []string) {
	skip := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		skip[col] = struct{}{}
	}
	eH.sync = nil
	eH.skip = skip
}

func (eH *BaseEventHandler) OnTableChanged(_ SourceInfo) error {
	// Nothing to do.
	return nil
}

func (eH *BaseEventHandler) OnRows(e *RowsEvent) ([]*Query, error) {
	switch e.Action {
	case ActionInsert:
		return eH.makeInsertBatch(e)
	case ActionUpdate:
		return eH.makeUpdateBatch(e)
	case ActionDelete:
		return eH.makeDeleteBatch(e)
	}

	return nil, fmt.Errorf("unknown rows action: %s", e.Action)
}

func (eH *BaseEventHandler) shouldSkip(name string) bool {
	skip := false

	if eH.skip != nil {
		_, skip = eH.skip[name]
	}

	if eH.sync != nil {
		_, sync := eH.sync[name]
		skip = !sync
	}

	return skip
}

func (eH *BaseEventHandler) makeInsertBatch(e *RowsEvent) ([]*Query, error) {
	queries := make([]*Query, 0, len(e.Rows))

	for _, row := range e.Rows {
		query, err := eH.makeInsertQuery(&e.Source, row)
		if err != nil {
			return nil, err
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func (eH *BaseEventHandler) makeInsertQuery(info *SourceInfo, row []interface{}) (*Query, error) {
	values := make([]QueryArg, 0, len(info.PKs))

	for _, pk := range info.PKs {
		arg, err := newQueryArg(pk, row)
		if err != nil {
			return nil, err
		}
		values = append(values, arg)
	}

	for _, col := range info.Cols {
		if eH.shouldSkip(col.Name) {
			continue
		}

		arg, err := newQueryArg(col, row)
		if err != nil {
			return nil, err
		}
		values = append(values, arg)
	}

	return &Query{
		Action: ActionInsert,
		Table:  eH.table,
		Values: values,
	}, nil
}

func (eH *BaseEventHandler) makeUpdateBatch(e *RowsEvent) ([]*Query, error) {
	if len(e.Rows)%2 != 0 {
		return nil, fmt.Errorf("invalid update rows event, must have 2x rows, but %d", len(e.Rows))
	}

	queries := make([]*Query, 0, len(e.Rows)/2)
	for i := 0; i < len(e.Rows); i += 2 {
		before := e.Rows[i]
		after := e.Rows[i+1]

		where := make([]QueryArg, 0, len(e.Source.PKs))
		values := make([]QueryArg, 0, len(e.Source.PKs))

		for _, pk := range e.Source.PKs {
			arg, err := newQueryArg(pk, before)
			if err != nil {
				return nil, err
			}
			where = append(where, arg)

			arg, err = newQueryArg(pk, after)
			if err != nil {
				return nil, err
			}
			values = append(values, arg)
		}

		for _, col := range e.Source.Cols {
			if eH.shouldSkip(col.Name) {
				continue
			}

			arg, err := newQueryArg(col, after)
			if err != nil {
				return nil, err
			}
			values = append(values, arg)
		}

		query := &Query{
			Action: ActionUpdate,
			Table:  eH.table,
			Values: values,
			Where:  where,
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func (eH *BaseEventHandler) makeDeleteBatch(e *RowsEvent) ([]*Query, error) {
	queries := make([]*Query, 0, len(e.Rows))

	for _, row := range e.Rows {
		where := make([]QueryArg, 0, len(e.Source.PKs))
		for _, pk := range e.Source.PKs {
			arg, err := newQueryArg(pk, row)
			if err != nil {
				return nil, err
			}
			where = append(where, arg)
		}

		query := &Query{
			Action: ActionDelete,
			Table:  eH.table,
			Where:  where,
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func newQueryArg(col Column, row []interface{}) (QueryArg, error) {
	value, err := col.GetValue(row)
	if err != nil {
		return QueryArg{}, err
	}

	return QueryArg{
		Field: col.Name,
		Value: value,
	}, nil
}
