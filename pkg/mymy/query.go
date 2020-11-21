package mymy

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmptyTable  = errors.New("query builder: empty table")
	ErrEmptyValues = errors.New("query builder: empty values")
	ErrEmptyWhere  = errors.New("query builder: empty where clause")
)

type Action string

const (
	ActionInsert Action = "insert"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type QueryArg struct {
	Field string
	Value interface{}
}

type Query struct {
	Action Action
	Table  string
	Values []QueryArg
	Where  []QueryArg
}

func (q *Query) SQL() (sql string, args []interface{}, err error) {
	switch q.Action {
	case ActionInsert:
		return q.toInsertSQL()
	case ActionUpdate:
		return q.toUpdateSQL()
	case ActionDelete:
		return q.toDeleteSQL()
	default:
		err = fmt.Errorf("unknown action type: %s", q.Action)
	}

	return
}

func (q *Query) toInsertSQL() (sql string, args []interface{}, err error) {
	if q.Table == "" {
		return "", nil, ErrEmptyTable
	}
	if len(q.Values) == 0 {
		return "", nil, ErrEmptyValues
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(q.Table)
	sb.WriteString(" (")
	for i, arg := range q.Values {
		sb.WriteString(arg.Field)
		if i < len(q.Values)-1 {
			sb.WriteRune(',')
		}
	}
	sb.WriteRune(')')
	sb.WriteString(" VALUES (")

	values := "?" + strings.Repeat(",?", len(q.Values)-1)
	sb.WriteString(values)
	sb.WriteRune(')')

	sql = sb.String()

	args = make([]interface{}, 0, len(values))
	for _, arg := range q.Values {
		args = append(args, arg.Value)
	}

	return sql, args, err
}

func (q *Query) toUpdateSQL() (sql string, args []interface{}, err error) {
	if q.Table == "" {
		return "", nil, ErrEmptyTable
	}
	if len(q.Values) == 0 {
		return "", nil, ErrEmptyValues
	}
	if len(q.Where) == 0 {
		return "", nil, ErrEmptyWhere
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(q.Table)
	sb.WriteString(" SET ")

	for i, arg := range q.Values {
		sb.WriteString(arg.Field)
		sb.WriteString("=?")
		if i < len(q.Values)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(" WHERE ")
	for i, arg := range q.Where {
		sb.WriteString(arg.Field)
		sb.WriteString("=?")
		if i < len(q.Where)-1 {
			sb.WriteString(", ")
		}
	}

	sql = sb.String()

	args = make([]interface{}, 0, len(q.Values)+len(q.Where))
	for _, arg := range q.Values {
		args = append(args, arg.Value)
	}
	for _, arg := range q.Where {
		args = append(args, arg.Value)
	}

	return sql, args, err
}

func (q *Query) toDeleteSQL() (sql string, args []interface{}, err error) {
	if q.Table == "" {
		return "", nil, ErrEmptyTable
	}
	if len(q.Where) == 0 {
		return "", nil, ErrEmptyWhere
	}

	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(q.Table)
	sb.WriteString(" WHERE ")

	for i, arg := range q.Where {
		sb.WriteString(arg.Field)
		sb.WriteString("=?")
		if i < len(q.Where)-1 {
			sb.WriteString(", ")
		}
	}

	sql = sb.String()

	args = make([]interface{}, 0, len(q.Where))
	for _, arg := range q.Where {
		args = append(args, arg.Value)
	}

	return sql, args, err
}
