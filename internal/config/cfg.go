//nolint:tagliatelle
package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultListenAddr         = ":8080"
	defaultDataFile           = "/etc/mymy/state.info"
	defaultPluginDir          = "plugins"
	defaultHealthSBM          = 10
	defaultLogLevel           = "debug"
	defaultSysLogEnabled      = false
	defaultFileLoggingEnabled = false
	defaultLogFilename        = "/var/log/mymy.log"
	defaultLogFileMaxSize     = 256
	defaultLogFileMaxBackups  = 3
	defaultLogFileMaxAge      = 5
	defaultMaxRetries         = 5
	defaultCharset            = "utf8mb4"
	defaultMaxOpenConns       = 200
	defaultMaxIdleConns       = 200
	defaultConnectTimeout     = 1 * time.Second
	defaultWriteTimeout       = 1 * time.Second
	defaultDumpSize           = 5000
	defaultLocalPathPrefix    = "bin/tmp"
	defaultDockerPathPrefix   = "/opt/dump"
)

type Config struct {
	App         AppConfig `yaml:"app"`
	Replication struct {
		// ServerID is the unique ID of the replica in MySQL cluster.
		// Omit this option if you'd like to auto generate ID.
		ServerID *uint32 `yaml:"server_id"`
		// GTIDMode indicates when to use GTID-based replication
		// or binlog file position.
		GTIDMode bool `yaml:"gtid_mode"`
		// SourceOpts is the options of leader.
		SourceOpts SourceConfig `yaml:"source"`
		// UpstreamOpts is the options of follower.
		UpstreamOpts UpstreamConfig `yaml:"upstream"`
		// Rules contains rules to handle data during the replication.
		Rules []RuleConfig `yaml:"rules"`
	} `yaml:"replication"`
}

type AppConfig struct {
	ListenAddr string  `yaml:"listen_addr"`
	DataFile   string  `yaml:"data_file"`
	PluginDir  string  `yaml:"plugin_dir"`
	Health     Health  `yaml:"health"`
	Logging    Logging `yaml:"logging"`
}

type Health struct {
	SecondsBehindMaster int `yaml:"seconds_behind_master"`
}

type Logging struct {
	Level              string `yaml:"level"`
	SysLogEnabled      bool   `yaml:"syslog_enabled"`
	FileLoggingEnabled bool   `yaml:"file_enabled"`
	Filename           string `yaml:"file_name"`
	MaxSize            int    `yaml:"file_max_size"`    // megabytes
	MaxBackups         int    `yaml:"file_max_backups"` // files
	MaxAge             int    `yaml:"file_max_age"`     // days
}

func (c *AppConfig) withDefaults() {
	if c == nil {
		return
	}

	c.ListenAddr = defaultListenAddr
	c.DataFile = defaultDataFile
	c.PluginDir = defaultPluginDir

	c.Health.SecondsBehindMaster = defaultHealthSBM

	c.Logging.Level = defaultLogLevel
	c.Logging.SysLogEnabled = defaultSysLogEnabled
	c.Logging.FileLoggingEnabled = defaultFileLoggingEnabled
	c.Logging.Filename = defaultLogFilename
	c.Logging.MaxSize = defaultLogFileMaxSize
	c.Logging.MaxBackups = defaultLogFileMaxBackups
	c.Logging.MaxAge = defaultLogFileMaxAge
}

type SourceConfig struct {
	Dump struct {
		// ExtraOptions for mysqldump CLI.
		ExtraOptions []string `yaml:"extra_options"`
		// ExecPath is absolute path to mysqldump binary.
		ExecPath string `yaml:"dump_exec_path"`
		// LocalPathDumpFile this is the path in the folder where files with requests for dump will be stored on the local machine.
		LocalPathDumpFile string `yaml:"local_path"`
		// DockerPathDumpFile this is the path in the folder where the files with requests for dump will be stored in the container.
		DockerPathDumpFile string `yaml:"docker_path"`
		// DumpSize is the storage size for the dump.
		DumpSize int `yaml:"dump_size"`
		// SkipMasterData set true if you have no privilege to use `--master-data`.
		SkipMasterData bool `yaml:"skip_master_data"`
		// DirectDump it is a flag that indicates whether to use load data statement for dump or to dump on separate requests.
		// Default false.
		DirectDump bool `yaml:"direct_dump"`
	} `yaml:"dump"`
	Addr     string `yaml:"addr"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
}

func (c *SourceConfig) withDefaults() {
	if c == nil {
		return
	}

	if c.Dump.ExecPath == "" {
		c.Dump.ExecPath = findDumpExecPath()
	}

	c.Charset = defaultCharset
	if c.Dump.DumpSize == 0 {
		c.Dump.DumpSize = defaultDumpSize
	}

	if c.Dump.LocalPathDumpFile == "" {
		c.Dump.LocalPathDumpFile = defaultLocalPathPrefix
	}

	if c.Dump.DockerPathDumpFile == "" {
		c.Dump.DockerPathDumpFile = defaultDockerPathPrefix
	}
}

func findDumpExecPath() string {
	findPath := []string{"/usr/bin/mysqldump", "/usr/local/bin/mysqldump"}

	for _, path := range findPath {
		if isFileExist(path) {
			return path
		}
	}

	return ""
}

func isFileExist(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	return true
}

type UpstreamConfig struct {
	Addr           string        `yaml:"addr"`
	User           string        `yaml:"user"`
	Password       string        `yaml:"password"`
	Database       string        `yaml:"database"`
	Charset        string        `yaml:"charset"`
	MaxRetries     int           `yaml:"max_retries"`
	MaxOpenConns   int           `yaml:"max_open_conns"`
	MaxIdleConns   int           `yaml:"max_idle_conns"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
}

func (c *UpstreamConfig) withDefaults() {
	if c == nil {
		return
	}

	c.MaxRetries = defaultMaxRetries
	c.MaxOpenConns = defaultMaxOpenConns
	c.MaxIdleConns = defaultMaxIdleConns
	c.ConnectTimeout = defaultConnectTimeout
	c.WriteTimeout = defaultWriteTimeout
}

type RuleConfig struct {
	Source struct {
		Table string `yaml:"table"`
	} `yaml:"source"`

	Upstream struct {
		Plugin struct {
			Name   string `yaml:"name"`
			Config string `yaml:"config"`
		} `yaml:"plugin"`
	} `yaml:"upstream"`
}

func ReadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	cfg.withDefaults()
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) withDefaults() {
	if c == nil {
		return
	}

	app := &c.App
	app.withDefaults()

	srcConn := &c.Replication.SourceOpts
	srcConn.withDefaults()

	destConn := &c.Replication.UpstreamOpts
	destConn.withDefaults()
}
