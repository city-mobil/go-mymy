package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/city-mobil/go-mymy/internal/adapter"
	"github.com/city-mobil/go-mymy/internal/bridge"
	"github.com/city-mobil/go-mymy/internal/client"
	"github.com/city-mobil/go-mymy/internal/config"
	"github.com/city-mobil/go-mymy/internal/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	sidlog "github.com/siddontang/go-log/log"
	"golang.org/x/sys/unix"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log/syslog"
	"os"
	"os/signal"
	"path"
	"syscall"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var (
	configPath = flag.String("config", "", "Config file path")
)

func main() {
	flag.Parse()
	cfg, err := config.ReadFromFile(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read config")
	}

	logger := initLogger(cfg)
	logger.Info().Msgf("starting replicator %s, commit %s, built at %s", version, commit, buildDate)

	factory := bridge.NewEventHandlerPluginFactory(cfg.App.PluginDir)
	b, err := bridge.New(cfg, factory, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not establish MySQL bridge")
	}

	sOpts := cfg.Replication.SourceOpts
	sClient, err := client.New(&client.Config{
		Addr:       sOpts.Addr,
		User:       sOpts.User,
		Password:   sOpts.Password,
		Database:   sOpts.Database,
		Charset:    sOpts.Charset,
		MaxRetries: 2,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error while creating client source base")
	}

	uOpts := cfg.Replication.UpstreamOpts
	uClient, err := client.New(&client.Config{
		Addr:           uOpts.Addr,
		User:           uOpts.User,
		Password:       uOpts.Password,
		Database:       uOpts.Database,
		Charset:        uOpts.Charset,
		MaxRetries:     uOpts.MaxRetries,
		MaxOpenConns:   uOpts.MaxOpenConns,
		MaxIdleConns:   uOpts.MaxIdleConns,
		ConnectTimeout: uOpts.ConnectTimeout,
		WriteTimeout:   uOpts.WriteTimeout,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error while creating client upstream base")
	}

	truncateBase(sClient, uClient, logger)

	rows := 2000

	// Prepare initial data.
	for i := 1; i <= rows; i++ {
		_, err = sClient.Exec(context.Background(), "INSERT INTO city.users (id, username, password, name, email) VALUES (?, ?, ?, ?, ?)", i, "bob", "12345", "Bob", "bob@email.com")
		if err != nil {
			logger.Fatal().Err(err).Msgf("error when filling the database with a request with a number %d", i)
		}
	}

	if !hasSyncedData(sClient, "city.users", rows) {
		logger.Fatal().Err(err).Msg("not enough data in the source database")
	}

	go func() {
		errRun := b.Run()
		if errRun != nil {
			logger.Err(errRun).Msg("got sync error")
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-interrupt

	logger.Info().Msgf("received system signal: %s. Shutting down replicator", sig)

	err = b.Close()
	if err != nil {
		logger.Err(err).Msg("got error on closing replicator")
	}

	if !hasSyncedData(uClient, "town.clients", rows) {
		logger.Fatal().Err(err).Msg("not enough data in the upstream database")
	}

	truncateBase(sClient, uClient, logger)
}

func initLogger(cfg *config.Config) zerolog.Logger {
	loggingCfg := cfg.App.Logging

	logLevel, err := zerolog.ParseLevel(loggingCfg.Level)
	if err != nil {
		log.Warn().Msgf("unknown Level string: '%s', defaulting to DebugLevel", loggingCfg.Level)
		logLevel = zerolog.DebugLevel
	}

	writers := make([]io.Writer, 0, 1)
	writers = append(writers, os.Stdout)

	if loggingCfg.SysLogEnabled {
		w, err := syslog.New(syslog.LOG_INFO, "mymy")
		if err != nil {
			log.Warn().Err(err).Msg("unable to connect to the system log daemon")
		} else {
			writers = append(writers, zerolog.SyslogLevelWriter(w))
		}
	}

	if loggingCfg.FileLoggingEnabled {
		w, err := newRollingLogFile(&loggingCfg)
		if err != nil {
			log.Warn().Err(err).Msg("unable to init file logger")
		} else {
			writers = append(writers, w)
		}
	}

	var baseLogger zerolog.Logger
	if len(writers) == 1 {
		baseLogger = zerolog.New(writers[0])
	} else {
		baseLogger = zerolog.New(zerolog.MultiLevelWriter(writers...))
	}

	logger := baseLogger.Level(logLevel).With().Timestamp().Logger()

	// Redirect siddontang/go-log messages to our logger.
	handler := adapter.NewZeroLogHandler(logger)
	sidlog.SetDefaultLogger(sidlog.New(handler, sidlog.Llevel))
	sidlog.SetLevelByName(logLevel.String())

	return logger
}

func newRollingLogFile(cfg *config.Logging) (io.Writer, error) {
	filepath := util.AbsPath(cfg.Filename)
	dir := path.Dir(filepath)
	if unix.Access(dir, unix.W_OK) != nil {
		return nil, fmt.Errorf("no permissions to write logs to dir: %s", dir)
	}

	return &lumberjack.Logger{
		Filename:   filepath,
		MaxBackups: cfg.MaxBackups,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
	}, nil
}

func truncateBase(sClient, uClient *client.SQLClient, logger zerolog.Logger) {
	_, err := sClient.Exec(context.Background(), "TRUNCATE city.users")
	if err != nil {
		logger.Fatal().Err(err).Msg("error while truncate source base")
	}

	_, err = uClient.Exec(context.Background(), "TRUNCATE town.clients")
	if err != nil {
		logger.Fatal().Err(err).Msg("error while truncate upstream base")
	}
}

func hasSyncedData(client *client.SQLClient, table string, rows int) bool {
	res := client.QueryRow(context.Background(), fmt.Sprintf("SELECT COUNT(*) FROM %s", table))

	var cnt int
	err := res.Scan(&cnt)
	if err != nil {
		return false
	}

	return cnt == rows
}
