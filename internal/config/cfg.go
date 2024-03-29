package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultListenAddr               = ":8080"
	defaultDataFile                 = "/etc/mymy/state.info"
	defaultPluginDir                = "plugins"
	defaultHealthSBM                = 10
	defaultLogLevel                 = "debug"
	defaultSysLogEnabled            = false
	defaultFileLoggingEnabled       = false
	defaultLogFilename              = "/var/log/mymy.log"
	defaultLogFileMaxSize           = 256
	defaultLogFileMaxBackups        = 3
	defaultLogFileMaxAge            = 5
	defaultMaxRetries               = 5
	defaultCharset                  = "utf8mb4"
	defaultMaxOpenConns             = 200
	defaultMaxIdleConns             = 200
	defaultConnectTimeout           = 1 * time.Second
	defaultWriteTimeout             = 1 * time.Second
	defaultLoadInFileFlushThreshold = 5000
	defaultАrgEnclose               = `"`
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
		ExecPath string `yaml:"exec_path"`
		// LoadInFileFlushThreshold defines a maximum number of rows to collect before flushing to the upstream database.
		LoadInFileFlushThreshold int `yaml:"load_in_file_flush_threshold"`
		// LoadInFileEnabled set true whether you want to dump the data using the LOAD DATA INFILE statement
		// instead of sending the transactions one by one.
		LoadInFileEnabled bool `yaml:"load_in_file_enabled"`
		// SkipMasterData set true if you have no privilege to use `--master-data`.
		SkipMasterData bool `yaml:"skip_master_data"`
		// ArgEnclose is a parameter that points to the beginning and end of the arguments in the dump file. Should be byte.
		ArgEnclose string `yaml:"arg_enclose"`
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

	c.Dump.ExecPath = findDumpExecPath()
	c.Charset = defaultCharset
	c.Dump.LoadInFileFlushThreshold = defaultLoadInFileFlushThreshold
	c.Dump.ArgEnclose = defaultАrgEnclose
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

	if cfg.Replication.SourceOpts.Dump.LoadInFileFlushThreshold == 0 {
		cfg.Replication.SourceOpts.Dump.LoadInFileFlushThreshold = defaultLoadInFileFlushThreshold
	}

	if len(cfg.Replication.SourceOpts.Dump.ArgEnclose) != 1 {
		cfg.Replication.SourceOpts.Dump.ArgEnclose = defaultАrgEnclose
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
