//nolint:paralleltest
package bridge

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/siddontang/go-mysql/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPosition_Clone(t *testing.T) {
	gtid, err := mysql.ParseMysqlGTIDSet("07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564")
	require.NoError(t, err)

	tests := []struct {
		name string
		pos  position
	}{
		{
			name: "GTID",
			pos:  newGTIDSet(gtid),
		},
		{
			name: "Binlog",
			pos: newBinlogPos(mysql.Position{
				Name: "mysql-bin.001650",
				Pos:  394877672,
			}),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pos.clone()
			assert.NotSame(t, tt.pos, got)
			assert.Equal(t, tt.pos, got)
		})
	}
}

func TestPosition_Marshal(t *testing.T) {
	gtid, err := mysql.ParseMysqlGTIDSet("07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564")
	require.NoError(t, err)

	tests := []struct {
		name string
		pos  position
		want string
	}{
		{
			name: "GTID",
			pos:  newGTIDSet(gtid),
			want: `{"gtid": "07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564"}`,
		},
		{
			name: "Binlog",
			pos: newBinlogPos(mysql.Position{
				Name: "mysql-bin.001650",
				Pos:  394877672,
			}),
			want: `{"name": "mysql-bin.001650", "pos": 394877672}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			buf, err := json.Marshal(tt.pos)
			require.NoError(t, err)

			assert.JSONEq(t, tt.want, string(buf))
		})
	}
}

func TestPosition_Unmarshal(t *testing.T) {
	gtid, err := mysql.ParseMysqlGTIDSet("07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564, 0d4becf8-5970-11ea-819f-1c34da0723b1:1-90")
	require.NoError(t, err)

	tests := []struct {
		name     string
		pos      string
		want     position
		gtidMode bool
	}{
		{
			name:     "GTID",
			pos:      `{"gtid": "07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564,0d4becf8-5970-11ea-819f-1c34da0723b1:1-90"}`,
			want:     newGTIDSet(gtid),
			gtidMode: true,
		},
		{
			name: "Binlog",
			pos:  `{"name": "mysql-bin.001650", "pos": 394877672}`,
			want: newBinlogPos(mysql.Position{
				Name: "mysql-bin.001650",
				Pos:  394877672,
			}),
			gtidMode: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got position
			if tt.gtidMode {
				got = &gtidSet{}
			} else {
				got = &binlogPos{}
			}

			err := json.Unmarshal([]byte(tt.pos), &got)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSaver_SaveLoad(t *testing.T) {
	oldGTID, err := mysql.ParseMysqlGTIDSet("07812e7f-5dad-11e6-b5b3-525400d2e382:1-939564")
	require.NoError(t, err)

	newGTID, err := mysql.ParseMysqlGTIDSet("07812e7f-5dad-11e6-b5b3-525400d2e382:1-939900")
	require.NoError(t, err)

	dataDir := "/tmp/mymy-save-test"
	dataFile := path.Join(dataDir, "master.info")

	tests := []struct {
		name     string
		oldPos   position
		newPos   position
		gtidMode bool
	}{
		{
			name:     "GTID",
			oldPos:   newGTIDSet(oldGTID),
			newPos:   newGTIDSet(newGTID),
			gtidMode: true,
		},
		{
			name: "Binlog",
			oldPos: newBinlogPos(mysql.Position{
				Name: "mysql-bin.001650",
				Pos:  394877672,
			}),
			newPos: newBinlogPos(mysql.Position{
				Name: "mysql-bin.001650",
				Pos:  394877900,
			}),
			gtidMode: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs, err := newFileSaver(dataFile, tt.gtidMode)
			if !assert.NoError(t, err) {
				return
			}
			// hack to imitate position changed event.
			fs.pos = tt.oldPos

			err = fs.save(tt.newPos, true)
			if !assert.NoError(t, err) {
				return
			}

			if !assert.FileExists(t, dataFile) {
				return
			}

			got, err := fs.load()
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.newPos, got)
		})

		err := os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}
}
