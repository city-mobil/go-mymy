package mymy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_SQL(t *testing.T) {
	type fields struct {
		Action Action
		Table  string
		Values []QueryArg
		Where  []QueryArg
	}
	tests := []struct {
		name     string
		fields   fields
		wantSQL  string
		wantArgs []interface{}
		wantErr  bool
	}{
		{
			name: "Insert_EmptyTable",
			fields: fields{
				Action: ActionInsert,
			},
			wantErr: true,
		},
		{
			name: "Insert_EmptyValues",
			fields: fields{
				Action: ActionInsert,
				Table:  "users",
			},
			wantErr: true,
		},
		{
			name: "Insert_OneValue",
			fields: fields{
				Action: ActionInsert,
				Table:  "users",
				Values: []QueryArg{
					{Field: "id", Value: 1},
				},
			},
			wantSQL:  "INSERT INTO users (id) VALUES (?)",
			wantArgs: []interface{}{1},
			wantErr:  false,
		},
		{
			name: "Insert_MultipleValues",
			fields: fields{
				Action: ActionInsert,
				Table:  "users",
				Values: []QueryArg{
					{Field: "id", Value: 1},
					{Field: "name", Value: "bob"},
					{Field: "email", Value: "bob@mail.com"},
				},
			},
			wantSQL:  "INSERT INTO users (id,name,email) VALUES (?,?,?)",
			wantArgs: []interface{}{1, "bob", "bob@mail.com"},
			wantErr:  false,
		},
		{
			name: "Update_EmptyTable",
			fields: fields{
				Action: ActionUpdate,
			},
			wantErr: true,
		},
		{
			name: "Update_EmptyWhere",
			fields: fields{
				Action: ActionUpdate,
				Values: []QueryArg{
					{Field: "id", Value: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "Update_EmptyValues",
			fields: fields{
				Action: ActionUpdate,
				Where: []QueryArg{
					{Field: "id", Value: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "Update_SimpleCase",
			fields: fields{
				Action: ActionUpdate,
				Table:  "users",
				Values: []QueryArg{
					{Field: "name", Value: "bob"},
				},
				Where: []QueryArg{
					{Field: "id", Value: 1},
				},
			},
			wantSQL:  "UPDATE users SET name=? WHERE id=?",
			wantArgs: []interface{}{"bob", 1},
			wantErr:  false,
		},
		{
			name: "Update_ComplexCase",
			fields: fields{
				Action: ActionUpdate,
				Table:  "users",
				Values: []QueryArg{
					{Field: "name", Value: "alice"},
					{Field: "email", Value: "bob@mail.com"},
				},
				Where: []QueryArg{
					{Field: "id", Value: 1},
					{Field: "name", Value: "bob"},
				},
			},
			wantSQL:  "UPDATE users SET name=?, email=? WHERE id=?, name=?",
			wantArgs: []interface{}{"alice", "bob@mail.com", 1, "bob"},
			wantErr:  false,
		},
		{
			name: "Delete_EmptyTable",
			fields: fields{
				Action: ActionDelete,
			},
			wantErr: true,
		},
		{
			name: "Delete_EmptyWhere",
			fields: fields{
				Action: ActionDelete,
				Table:  "users",
			},
			wantErr: true,
		},
		{
			name: "Delete_OneWhere",
			fields: fields{
				Action: ActionDelete,
				Table:  "users",
				Where: []QueryArg{
					{Field: "id", Value: 1},
				},
			},
			wantSQL:  "DELETE FROM users WHERE id=?",
			wantArgs: []interface{}{1},
			wantErr:  false,
		},
		{
			name: "Delete_ComplexWhereClause",
			fields: fields{
				Action: ActionDelete,
				Table:  "users",
				Where: []QueryArg{
					{Field: "name", Value: "bob"},
					{Field: "email", Value: "bob@mail.com"},
				},
			},
			wantSQL:  "DELETE FROM users WHERE name=?, email=?",
			wantArgs: []interface{}{"bob", "bob@mail.com"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			q := &Query{
				Action: tt.fields.Action,
				Table:  tt.fields.Table,
				Values: tt.fields.Values,
				Where:  tt.fields.Where,
			}
			gotSQL, gotArgs, err := q.SQL()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, gotSQL)
				assert.Empty(t, gotArgs)
			} else {
				assert.Equal(t, tt.wantSQL, gotSQL)
				assert.EqualValues(t, tt.wantArgs, gotArgs)
			}
		})
	}
}
