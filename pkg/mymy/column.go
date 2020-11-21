package mymy

import (
	"fmt"
)

type ColumnType int

const (
	TypeNumber    ColumnType = iota + 1 // tinyint, smallint, int, bigint, year
	TypeFloat                           // float, double
	TypeEnum                            // enum
	TypeSet                             // set
	TypeString                          // char, varchar, etc.
	TypeDatetime                        // datetime
	TypeTimestamp                       // timestamp
	TypeDate                            // date
	TypeTime                            // time
	TypeBit                             // bit
	TypeJSON                            // json
	TypeDecimal                         // decimal
	TypeMediumInt                       // medium int
	TypeBinary                          // binary, varbinary
	TypePoint                           // coordinates
)

type Column struct {
	Index      uint64
	Name       string
	Type       ColumnType
	Collation  string
	IsAuto     bool
	IsUnsigned bool
	IsVirtual  bool
}

func (c *Column) GetValue(row []interface{}) (interface{}, error) {
	if c.Index >= uint64(len(row)) {
		return nil, fmt.Errorf("column index (%d) equals or greater than row length (%d)", c.Index, len(row))
	}

	return row[c.Index], nil
}
