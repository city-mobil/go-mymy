package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/etherlabsio/healthcheck"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	sidlog "github.com/siddontang/go-log/log"
	"golang.org/x/sys/unix"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/city-mobil/go-mymy/internal/adapter"
	"github.com/city-mobil/go-mymy/internal/bridge"
	"github.com/city-mobil/go-mymy/internal/config"
	"github.com/city-mobil/go-mymy/internal/metrics"
	"github.com/city-mobil/go-mymy/internal/util"

	_ "github.com/go-sql-driver/mysql"
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

	metrics.Init()

	factory := bridge.NewEventHandlerPluginFactory(cfg.App.PluginDir)
	b, err := bridge.New(cfg, factory, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not establish MySQL bridge")
	}

	healthHd := initHealthHandler(cfg.App.Health, b)
	aboutHd := initAboutHandler(version, commit, buildDate)
	server := initHTTPServer(cfg.App.ListenAddr, healthHd, aboutHd)
	go func() {
		logger.Info().Msgf("listening on %s", cfg.App.ListenAddr)

		err = server.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("failed to listen HTTP server")
		}
	}()

	go func() {
		errRun := b.Run()
		if errRun != nil {
			logger.Err(errRun).Msg("got sync error")
		}

		errClose := b.Close()
		if errClose != nil {
			logger.Err(errClose).Msg("got error on closing replicator")
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

	err = server.Shutdown(context.Background())
	if err != nil {
		logger.Err(err).Msg("failed to shutting down the HTTP server gracefully")
	}
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

func initHTTPServer(addr string, healthHd, aboutHd http.Handler) *http.Server {
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/health", healthHd)
	http.Handle("/about", aboutHd)

	return server
}

func initHealthHandler(cfg config.Health, b *bridge.Bridge) http.Handler {
	sbm := uint32(cfg.SecondsBehindMaster)

	return healthcheck.Handler(
		healthcheck.WithChecker(
			"lag", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					cur := b.Delay()
					if cur > sbm {
						return fmt.Errorf("replication lag too big: %d", cur)
					}

					return nil
				},
			),
		),

		healthcheck.WithChecker(
			"state", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					dumping := b.Dumping()
					if dumping {
						return errors.New("replicator has not yet finished dump process")
					}

					running := b.Running()
					if !running {
						return errors.New("replication is not running")
					}

					return nil
				},
			),
		),
	)
}

func initAboutHandler(version, commit, buildDate string) http.Handler {
	about := struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Build   string `json:"build"`
	}{
		Version: version,
		Commit:  commit,
		Build:   buildDate,
	}

	aboutStr, _ := json.Marshal(about)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(aboutStr)
	})
}
