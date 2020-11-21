//nolint:unused,deadcode
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/city-mobil/go-mymy/pkg/mymy"
)

type config struct {
	Table   string   `yaml:"table"`
	Columns []string `yaml:"columns"`
}

func readConfig(path string) (*config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

type FilterEventHandler struct {
	table  string
	filter map[string]struct{}
}

func NewEventHandler(cfgPath string) (mymy.EventHandler, error) {
	cfg, err := readConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	filter := make(map[string]struct{}, len(cfg.Columns))
	for _, col := range cfg.Columns {
		filter[col] = struct{}{}
	}

	return &FilterEventHandler{
		table:  cfg.Table,
		filter: filter,
	}, nil
}

func (eH *FilterEventHandler) OnRows(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	switch e.Action {
	case mymy.ActionInsert:
		return eH.makeInsertBatch(e)
	case mymy.ActionUpdate:
		return eH.makeUpdateBatch(e)
	case mymy.ActionDelete:
		return eH.makeDeleteBatch(e)
	}

	return nil, fmt.Errorf("unknown rows action: %s", e.Action)
}

func (eH *FilterEventHandler) shouldSync(name string) bool {
	_, ok := eH.filter[name]

	return ok
}

func (eH *FilterEventHandler) makeInsertBatch(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	queries := make([]*mymy.Query, 0, len(e.Rows))

	for _, row := range e.Rows {
		query, err := eH.makeInsertQuery(&e.Source, row)
		if err != nil {
			return nil, err
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func (eH *FilterEventHandler) makeInsertQuery(info *mymy.SourceInfo, row []interface{}) (*mymy.Query, error) {
	values := make([]mymy.QueryArg, 0, len(info.PKs))

	for _, pk := range info.PKs {
		arg, err := newQueryArg(pk, row)
		if err != nil {
			return nil, err
		}
		values = append(values, arg)
	}

	for _, col := range info.Cols {
		if !eH.shouldSync(col.Name) {
			continue
		}

		arg, err := newQueryArg(col, row)
		if err != nil {
			return nil, err
		}
		values = append(values, arg)
	}

	return &mymy.Query{
		Action: mymy.ActionInsert,
		Table:  eH.table,
		Values: values,
	}, nil
}

func (eH *FilterEventHandler) makeUpdateBatch(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	if len(e.Rows)%2 != 0 {
		return nil, fmt.Errorf("invalid update rows event, must have 2x rows, but %d", len(e.Rows))
	}

	queries := make([]*mymy.Query, 0, len(e.Rows)/2)
	for i := 0; i < len(e.Rows); i += 2 {
		before := e.Rows[i]
		after := e.Rows[i+1]

		where := make([]mymy.QueryArg, 0, len(e.Source.PKs))
		values := make([]mymy.QueryArg, 0, len(e.Source.PKs))

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
			if !eH.shouldSync(col.Name) {
				continue
			}

			arg, err := newQueryArg(col, after)
			if err != nil {
				return nil, err
			}
			values = append(values, arg)
		}

		query := &mymy.Query{
			Action: mymy.ActionUpdate,
			Table:  eH.table,
			Values: values,
			Where:  where,
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func (eH *FilterEventHandler) makeDeleteBatch(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	queries := make([]*mymy.Query, 0, len(e.Rows))

	for _, row := range e.Rows {
		where := make([]mymy.QueryArg, 0, len(e.Source.PKs))
		for _, pk := range e.Source.PKs {
			arg, err := newQueryArg(pk, row)
			if err != nil {
				return nil, err
			}
			where = append(where, arg)
		}

		query := &mymy.Query{
			Action: mymy.ActionDelete,
			Table:  eH.table,
			Where:  where,
		}

		queries = append(queries, query)
	}

	return queries, nil
}

func newQueryArg(col mymy.Column, row []interface{}) (mymy.QueryArg, error) {
	value, err := col.GetValue(row)
	if err != nil {
		return mymy.QueryArg{}, err
	}

	return mymy.QueryArg{
		Field: col.Name,
		Value: value,
	}, nil
}
