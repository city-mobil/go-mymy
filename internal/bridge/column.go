package bridge

import (
	"github.com/siddontang/go-mysql/schema"

	"github.com/city-mobil/go-mymy/pkg/mymy"
)

func newColumn(idx int, src *schema.TableColumn) mymy.Column {
	return mymy.Column{
		Index:      uint64(idx),
		Name:       src.Name,
		Type:       mymy.ColumnType(src.Type),
		Collation:  src.Collation,
		IsUnsigned: src.IsUnsigned,
		IsAuto:     src.IsAuto,
		IsVirtual:  src.IsVirtual,
	}
}

func newColumnsFromNonPKs(table *schema.Table) []mymy.Column {
	cols := make([]mymy.Column, 0, len(table.Columns))
	for idx := range table.Columns {
		col := &table.Columns[idx]

		isPK := false
		for _, pkIdx := range table.PKColumns {
			if idx == pkIdx {
				isPK = true

				break
			}
		}

		if !isPK {
			cols = append(cols, newColumn(idx, col))
		}
	}

	return cols
}

func newColumnsFromPKs(table *schema.Table) []mymy.Column {
	pks := make([]mymy.Column, 0)
	for _, idx := range table.PKColumns {
		col := table.GetPKColumn(idx)
		pks = append(pks, newColumn(idx, col))
	}

	return pks
}
