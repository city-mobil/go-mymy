package mymy

import "strings"

type SourceInfo struct {
	Schema string
	Table  string
	PKs    []Column
	Cols   []Column
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
