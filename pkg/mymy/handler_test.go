package mymy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	tSource = SourceInfo{
		Schema: "city",
		Table:  "clients",
		PKs: []Column{
			{
				Index:      0,
				Name:       "id",
				Type:       TypeNumber,
				IsAuto:     true,
				IsUnsigned: true,
			},
		},
		Cols: []Column{
			{Index: 1, Name: "name", Type: TypeString},
			{Index: 2, Name: "email", Type: TypeString},
			{Index: 3, Name: "position", Type: TypeString},
		},
	}
)

func TestBaseEventHandler_OnRows(t *testing.T) {
	type fields struct {
		table string
		sync  []string
	}
	type args struct {
		e *RowsEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*Query
		wantErr bool
	}{
		{
			name: "UnknownAction",
			fields: fields{
				table: "users",
				sync:  []string{"name", "email"},
			},
			args: args{
				e: &RowsEvent{
					Action: Action("upsert"),
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
					},
				},
			},
			wantErr: true,
		},

		{
			name: "OnInsert",
			fields: fields{
				table: "users",
				sync:  []string{"name", "email"},
			},
			args: args{
				e: &RowsEvent{
					Action: ActionInsert,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{2, "alice", "alice@mail.com", "CEO"},
					},
				},
			},
			want: []*Query{
				{
					Action: ActionInsert,
					Table:  "users",
					Values: []QueryArg{
						{Field: "id", Value: 1},
						{Field: "name", Value: "bob"},
						{Field: "email", Value: "bob@mail.com"},
					},
				},
				{
					Action: ActionInsert,
					Table:  "users",
					Values: []QueryArg{
						{Field: "id", Value: 2},
						{Field: "name", Value: "alice"},
						{Field: "email", Value: "alice@mail.com"},
					},
				},
			},
			wantErr: false,
		},

		{
			name: "OnUpdate",
			fields: fields{
				table: "users",
				sync:  []string{"name", "email"},
			},
			args: args{
				e: &RowsEvent{
					Action: ActionUpdate,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{1, "alice", "alice@mail.com", "CEO"},
						{2, "john", "john@mail.com", "Lifter"},
						{3, "john", "john@mail.com", "CEO"},
					},
				},
			},
			want: []*Query{
				{
					Action: ActionUpdate,
					Table:  "users",
					Values: []QueryArg{
						{Field: "id", Value: 1},
						{Field: "name", Value: "alice"},
						{Field: "email", Value: "alice@mail.com"},
					},
					Where: []QueryArg{
						{Field: "id", Value: 1},
					},
				},
				{
					Action: ActionUpdate,
					Table:  "users",
					Values: []QueryArg{
						{Field: "id", Value: 3},
						{Field: "name", Value: "john"},
						{Field: "email", Value: "john@mail.com"},
					},
					Where: []QueryArg{
						{Field: "id", Value: 2},
					},
				},
			},
			wantErr: false,
		},

		{
			name: "OnDelete",
			fields: fields{
				table: "users",
				sync:  []string{"name", "email"},
			},
			args: args{
				e: &RowsEvent{
					Action: ActionDelete,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{2, "alice", "alice@mail.com", "CEO"},
					},
				},
			},
			want: []*Query{
				{
					Action: ActionDelete,
					Table:  "users",
					Where: []QueryArg{
						{Field: "id", Value: 1},
					},
				},
				{
					Action: ActionDelete,
					Table:  "users",
					Where: []QueryArg{
						{Field: "id", Value: 2},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			eH := NewBaseEventHandler(tt.fields.table)
			eH.SyncOnly(tt.fields.sync)

			got, err := eH.OnRows(tt.args.e)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}
