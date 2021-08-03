package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/city-mobil/go-mymy/internal/bridge"
	"github.com/city-mobil/go-mymy/internal/client"
	"github.com/city-mobil/go-mymy/internal/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configPath = flag.String("config", "", "Config file path")
)

var (
	logger zerolog.Logger
)

const (
	sourceRows = 20000
)

func main() {
	flag.Parse()
	cfg, err := config.ReadFromFile(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read config")
	}

	logger = zerolog.New(os.Stdout).Level(zerolog.DebugLevel).With().Timestamp().Logger()

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
		logger.Fatal().Err(err).Msg("could not connect to source db")
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
		logger.Fatal().Err(err).Msg("could not connect to upstream db")
	}

	initDBs(sClient, uClient)
	defer truncateTables(sClient, uClient)

	start := time.Now()
	go func() {
		<-b.WaitDumpDone()

		end := time.Since(start)
		logger.Info().Msgf("dump finished in %d ms", end.Milliseconds())

		cerr := b.Close()
		if cerr != nil {
			logger.Error().Err(cerr).Msg("got error on closing replicator")
		}
	}()

	err = b.Run()
	if err != nil {
		logger.Error().Err(err).Msg("got sync error")
	}

	if !hasSyncedData(uClient) {
		logger.Error().Err(err).Msg("not enough data in the upstream database")
	}
}

func initDBs(sClient, uClient *client.SQLClient) {
	truncateTables(sClient, uClient)

	query := "INSERT INTO city.users (id, username, password, name, email) VALUES (?, ?, ?, ?, ?)"
	for i := 1; i <= sourceRows; i++ {
		_, err := sClient.Exec(context.Background(), query, i, "bob", "12345", "Bob", "bob@email.com")
		if err != nil {
			logger.Fatal().Err(err).Msgf("could not insert the row №%d", i)
		}
	}
}

func truncateTables(sClient, uClient *client.SQLClient) {
	_, err := sClient.Exec(context.Background(), "TRUNCATE city.users")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not truncate source table")
	}

	_, err = uClient.Exec(context.Background(), "TRUNCATE town.clients")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not truncate upstream table")
	}
}

func hasSyncedData(upstream *client.SQLClient) bool {
	res := upstream.QueryRow(context.Background(), "SELECT COUNT(*) FROM town.clients")

	var cnt int
	if err := res.Scan(&cnt); err != nil {
		log.Fatal().Err(err).Msg("could not fetch rows count in upstream")
	}

	return cnt == sourceRows
}
