package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "AllOptions",
			cfg: &Config{
				Addr:           "db1.storage.ru:3306",
				User:           "admin",
				Password:       "admin1",
				Database:       "meta",
				Charset:        "utf8",
				ConnectTimeout: 2 * time.Minute,
				WriteTimeout:   100 * time.Millisecond,
			},
			want: "admin:admin1@tcp(db1.storage.ru:3306)/meta?interpolateParams=true&timeout=2m0s&writeTimeout=100ms&charset=utf8",
		},
		{
			name: "NoCharset",
			cfg: &Config{
				Addr:           "db1.storage.ru:3306",
				User:           "admin",
				Password:       "admin1",
				Database:       "meta",
				ConnectTimeout: 2 * time.Minute,
				WriteTimeout:   100 * time.Millisecond,
			},
			want: "admin:admin1@tcp(db1.storage.ru:3306)/meta?interpolateParams=true&timeout=2m0s&writeTimeout=100ms",
		},
		{
			name: "NoTimeouts",
			cfg: &Config{
				Addr:     "db1.storage.ru:3306",
				User:     "admin",
				Password: "admin1",
				Database: "meta",
			},
			want: "admin:admin1@tcp(db1.storage.ru:3306)/meta?interpolateParams=true",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.DSN()
			assert.Equal(t, tt.want, got)
		})
	}
}
