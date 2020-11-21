package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromFile_InvalidPath(t *testing.T) {
	cfg, err := ReadFromFile("invalid_path")
	assert.NotNil(t, err)
	assert.Nil(t, cfg)
}

func TestReadFromFile_ValidPath(t *testing.T) {
	testConfigPath, err := filepath.Abs("testdata/mymy.yml")
	require.NoError(t, err)

	cfg, err := ReadFromFile(testConfigPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, ":8081", cfg.App.ListenAddr)
	assert.Equal(t, "/etc/mymy/state.info", cfg.App.DataFile)
	assert.Equal(t, "/etc/mymy/plugins", cfg.App.PluginDir)

	healthCfg := cfg.App.Health
	assert.Equal(t, 5, healthCfg.SecondsBehindMaster)

	loggingCfg := cfg.App.Logging
	assert.Equal(t, "debug", loggingCfg.Level)
	assert.True(t, loggingCfg.SysLogEnabled)
	assert.True(t, loggingCfg.FileLoggingEnabled)
	assert.Equal(t, "/var/log/mymy.log", loggingCfg.Filename)
	assert.Equal(t, 256, loggingCfg.MaxSize)
	assert.Equal(t, 3, loggingCfg.MaxBackups)
	assert.Equal(t, 5, loggingCfg.MaxAge)

	require.NotNil(t, cfg.Replication.ServerID)
	assert.EqualValues(t, 100, *cfg.Replication.ServerID)
	assert.True(t, cfg.Replication.GTIDMode)

	source := cfg.Replication.SourceOpts
	assert.Equal(t, "/usr/bin/mysqldump", source.Dump.ExecPath)
	assert.False(t, source.Dump.SkipMasterData)
	assert.Equal(t, []string{"--column-statistics=0"}, source.Dump.ExtraOptions)
	assert.Equal(t, "127.0.0.1:3306", source.Addr)
	assert.Equal(t, "repl", source.User)
	assert.Equal(t, "repl", source.Password)
	assert.Equal(t, "city", source.Database)
	assert.Equal(t, "utf8", source.Charset)

	upstream := cfg.Replication.UpstreamOpts
	assert.Equal(t, "127.0.0.1:3307", upstream.Addr)
	assert.Equal(t, "repl", upstream.User)
	assert.Equal(t, "repl", upstream.Password)
	assert.Equal(t, "town", upstream.Database)
	assert.Equal(t, 3, upstream.MaxRetries)
	assert.Equal(t, 500, upstream.MaxOpenConns)
	assert.Equal(t, 500, upstream.MaxIdleConns)
	assert.Equal(t, 500*time.Millisecond, upstream.ConnectTimeout)
	assert.Equal(t, 500*time.Millisecond, upstream.WriteTimeout)

	rules := cfg.Replication.Rules
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "users", rule.Source.Table)
	assert.Equal(t, "mymy_filter", rule.Upstream.Plugin.Name)
	assert.Equal(t, "plugins/mymy.filter.yml", rule.Upstream.Plugin.Config)
}

func TestUpstreamConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  *UpstreamConfig
		want string
	}{
		{
			name: "AllOptions",
			cfg: &UpstreamConfig{
				Addr:           "db1.storage.ru:3306",
				User:           "admin",
				Password:       "admin1",
				Database:       "meta",
				Charset:        "utf8",
				ConnectTimeout: 2 * time.Minute,
				WriteTimeout:   100 * time.Millisecond,
			},
			want: "admin:admin1@tcp(db1.storage.ru:3306)/meta?timeout=2m0s&writeTimeout=100ms&interpolateParams=true&charset=utf8",
		},
		{
			name: "NoCharset",
			cfg: &UpstreamConfig{
				Addr:           "db1.storage.ru:3306",
				User:           "admin",
				Password:       "admin1",
				Database:       "meta",
				ConnectTimeout: 2 * time.Minute,
				WriteTimeout:   100 * time.Millisecond,
			},
			want: "admin:admin1@tcp(db1.storage.ru:3306)/meta?timeout=2m0s&writeTimeout=100ms&interpolateParams=true",
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
