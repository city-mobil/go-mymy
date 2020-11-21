package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/city-mobil/go-mymy/pkg/mymy"
)

var (
	tSource = mymy.SourceInfo{
		Schema: "city",
		Table:  "clients",
		PKs: []mymy.Column{
			{
				Index:      0,
				Name:       "id",
				Type:       mymy.TypeNumber,
				IsAuto:     true,
				IsUnsigned: true,
			},
		},
		Cols: []mymy.Column{
			{Index: 1, Name: "name", Type: mymy.TypeString},
			{Index: 2, Name: "email", Type: mymy.TypeString},
			{Index: 3, Name: "position", Type: mymy.TypeString},
		},
	}
)

func TestFilterEventHandler_OnRows(t *testing.T) {
	type fields struct {
		table  string
		filter map[string]struct{}
	}
	type args struct {
		e *mymy.RowsEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*mymy.Query
		wantErr bool
	}{
		{
			name: "UnknownAction",
			fields: fields{
				table: "users",
				filter: map[string]struct{}{
					"name":  {},
					"email": {},
				},
			},
			args: args{
				e: &mymy.RowsEvent{
					Action: mymy.Action("upsert"),
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
				filter: map[string]struct{}{
					"name":  {},
					"email": {},
				},
			},
			args: args{
				e: &mymy.RowsEvent{
					Action: mymy.ActionInsert,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{2, "alice", "alice@mail.com", "CEO"},
					},
				},
			},
			want: []*mymy.Query{
				{
					Action: mymy.ActionInsert,
					Table:  "users",
					Values: []mymy.QueryArg{
						{Field: "id", Value: 1},
						{Field: "name", Value: "bob"},
						{Field: "email", Value: "bob@mail.com"},
					},
				},
				{
					Action: mymy.ActionInsert,
					Table:  "users",
					Values: []mymy.QueryArg{
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
				filter: map[string]struct{}{
					"name":  {},
					"email": {},
				},
			},
			args: args{
				e: &mymy.RowsEvent{
					Action: mymy.ActionUpdate,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{1, "alice", "alice@mail.com", "CEO"},
						{2, "john", "john@mail.com", "Lifter"},
						{3, "john", "john@mail.com", "CEO"},
					},
				},
			},
			want: []*mymy.Query{
				{
					Action: mymy.ActionUpdate,
					Table:  "users",
					Values: []mymy.QueryArg{
						{Field: "id", Value: 1},
						{Field: "name", Value: "alice"},
						{Field: "email", Value: "alice@mail.com"},
					},
					Where: []mymy.QueryArg{
						{Field: "id", Value: 1},
					},
				},
				{
					Action: mymy.ActionUpdate,
					Table:  "users",
					Values: []mymy.QueryArg{
						{Field: "id", Value: 3},
						{Field: "name", Value: "john"},
						{Field: "email", Value: "john@mail.com"},
					},
					Where: []mymy.QueryArg{
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
				filter: map[string]struct{}{
					"name":  {},
					"email": {},
				},
			},
			args: args{
				e: &mymy.RowsEvent{
					Action: mymy.ActionDelete,
					Source: tSource,
					Rows: [][]interface{}{
						{1, "bob", "bob@mail.com", "CTO"},
						{2, "alice", "alice@mail.com", "CEO"},
					},
				},
			},
			want: []*mymy.Query{
				{
					Action: mymy.ActionDelete,
					Table:  "users",
					Where: []mymy.QueryArg{
						{Field: "id", Value: 1},
					},
				},
				{
					Action: mymy.ActionDelete,
					Table:  "users",
					Where: []mymy.QueryArg{
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
			eH := &FilterEventHandler{
				table:  tt.fields.table,
				filter: tt.fields.filter,
			}
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
