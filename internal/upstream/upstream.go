package upstream

import (
	"context"
	"database/sql"

	"github.com/city-mobil/go-mymy/internal/config"
)

const (
	defaultMaxRetries = 5
)

type Upstream struct {
	db      *sql.DB
	retries int
}

func New(cfg *config.UpstreamConfig) (*Upstream, error) {
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

	return &Upstream{
		db:      db,
		retries: retries,
	}, nil
}

func (u *Upstream) Exec(ctx context.Context, query string, args ...interface{}) (res sql.Result, err error) {
	for attempt := 0; attempt <= u.retries; attempt++ {
		res, err = u.db.ExecContext(ctx, query, args...)
		if canRetry(err) {
			continue
		}

		break
	}

	return
}

func (u *Upstream) Close() error {
	return u.db.Close()
}
