//nolint:paralleltest
package mymy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceInfo_FindColumnByName(t *testing.T) {
	tests := []struct {
		name    string
		info    SourceInfo
		lookup  string
		want    Column
		wantErr bool
	}{
		{
			name: "PrimaryKey",
			info: SourceInfo{
				Schema: "city",
				Table:  "users",
				PKs: []Column{
					{
						Index:  0,
						Name:   "id",
						Type:   TypeNumber,
						IsAuto: true,
					},
				},
				Cols: []Column{
					{
						Index: 1,
						Name:  "name",
						Type:  TypeString,
					},
					{
						Index: 2,
						Name:  "password",
						Type:  TypeString,
					},
				},
			},
			lookup: "id",
			want: Column{
				Index:  0,
				Name:   "id",
				Type:   TypeNumber,
				IsAuto: true,
			},
			wantErr: false,
		},
		{
			name: "SimpleColumn",
			info: SourceInfo{
				Schema: "city",
				Table:  "users",
				PKs: []Column{
					{
						Index:  0,
						Name:   "id",
						Type:   TypeNumber,
						IsAuto: true,
					},
				},
				Cols: []Column{
					{
						Index: 1,
						Name:  "name",
						Type:  TypeString,
					},
					{
						Index: 2,
						Name:  "password",
						Type:  TypeString,
					},
				},
			},
			lookup: "password",
			want: Column{
				Index: 2,
				Name:  "password",
				Type:  TypeString,
			},
			wantErr: false,
		},
		{
			name: "NotFound",
			info: SourceInfo{
				Schema: "city",
				Table:  "users",
				PKs: []Column{
					{
						Index:  0,
						Name:   "id",
						Type:   TypeNumber,
						IsAuto: true,
					},
				},
				Cols: []Column{
					{
						Index: 1,
						Name:  "name",
						Type:  TypeString,
					},
					{
						Index: 2,
						Name:  "password",
						Type:  TypeString,
					},
				},
			},
			lookup:  "alias",
			want:    Column{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.info.FindColumnByName(tt.lookup)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, ErrColumnNotFound.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
