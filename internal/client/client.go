package client

import (
	"context"
	"database/sql"
)

const (
	defaultMaxRetries = 5
)

type SQLClient struct {
	db      *sql.DB
	retries int
}

func New(cfg *Config) (*SQLClient, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	retries := cfg.MaxRetries
	if retries < 0 {
		retries = defaultMaxRetries
	}

	return &SQLClient{
		db:      db,
		retries: retries,
	}, nil
}

func (c *SQLClient) Exec(ctx context.Context, query string, args ...interface{}) (res sql.Result, err error) {
	for attempt := 0; attempt <= c.retries; attempt++ {
		res, err = c.db.ExecContext(ctx, query, args...)
		if canRetry(err) {
			continue
		}

		break
	}

	return
}

func (c *SQLClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

func (c *SQLClient) Close() error {
	return c.db.Close()
}
