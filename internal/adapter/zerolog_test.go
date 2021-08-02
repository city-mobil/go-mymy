//nolint:paralleltest
package adapter

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_parseLevelAndMsg(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		wantLvl zerolog.Level
		wantMsg string
	}{
		{
			name:    "InfoMessage",
			arg:     []byte("[info] message"),
			wantLvl: zerolog.InfoLevel,
			wantMsg: "message",
		},
		{
			name:    "ErrorMessage",
			arg:     []byte("[error] message"),
			wantLvl: zerolog.ErrorLevel,
			wantMsg: "message",
		},
		{
			name:    "UnknownLevel",
			arg:     []byte("[mylevel] message"),
			wantLvl: zerolog.InfoLevel,
			wantMsg: "[mylevel] message",
		},
		{
			name:    "OnlyLevel",
			arg:     []byte("[info]"),
			wantLvl: zerolog.InfoLevel,
			wantMsg: "",
		},
		{
			name:    "BadMessageFormat",
			arg:     []byte("[info message"),
			wantLvl: zerolog.InfoLevel,
			wantMsg: "[info message",
		},
		{
			name:    "EmptyMessage",
			arg:     []byte(""),
			wantLvl: zerolog.InfoLevel,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			level, msg := parseLevelAndMsg(tt.arg)
			assert.Equal(t, tt.wantLvl, level)
			assert.Equal(t, tt.wantMsg, msg)
		})
	}
}
