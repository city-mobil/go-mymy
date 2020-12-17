package mymy

import (
	"errors"
	"strings"
)

var (
	ErrColumnNotFound = errors.New("column not found")
)

type SourceInfo struct {
	Schema string
	Table  string
	PKs    []Column
	Cols   []Column
}

func (info SourceInfo) FindColumnByName(name string) (Column, error) {
	for _, col := range info.PKs {
		if col.Name == name {
			return col, nil
		}
	}

	for _, col := range info.Cols {
		if col.Name == name {
			return col, nil
		}
	}

	return Column{}, ErrColumnNotFound
}

type Rule struct {
	Source  SourceInfo
	Handler EventHandler
}

func RuleKey(schema, table string) string {
	var sb strings.Builder
	sb.Grow(len(schema) + len(table) + 1)
	sb.WriteString(schema)
	sb.WriteByte(':')
	sb.WriteString(table)

	return sb.String()
}
