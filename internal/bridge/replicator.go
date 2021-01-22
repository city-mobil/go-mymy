package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"go.uber.org/atomic"
	"go.uber.org/multierr"

	"github.com/city-mobil/go-mymy/internal/client"
	"github.com/city-mobil/go-mymy/internal/config"
	"github.com/city-mobil/go-mymy/internal/metrics"
	"github.com/city-mobil/go-mymy/pkg/mymy"
)

const eventsBufSize = 4096

var ErrRuleNotExist = errors.New("rule is not exist")

type batch []*mymy.Query

type Bridge struct {
	rules map[string]*mymy.Rule

	canal      *canal.Canal
	upstream   *client.SQLClient
	stateSaver stateSaver

	ctx    context.Context
	cancel context.CancelFunc
	logger zerolog.Logger

	dumping  *atomic.Bool
	running  *atomic.Bool
	syncedAt *atomic.Int64

	syncCh    chan interface{}
	closeOnce *sync.Once
}

func New(cfg *config.Config, ehFactory EventHandlerFactory, logger zerolog.Logger) (*Bridge, error) {
	b := &Bridge{
		logger:    logger,
		dumping:   atomic.NewBool(false),
		running:   atomic.NewBool(false),
		syncedAt:  atomic.NewInt64(0),
		syncCh:    make(chan interface{}, eventsBufSize),
		closeOnce: &sync.Once{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.ctx = ctx
	b.cancel = cancel

	if err := b.newStateSaver(cfg); err != nil {
		return nil, err
	}

	if err := b.newCanal(cfg); err != nil {
		return nil, err
	}

	if err := b.newRules(cfg, ehFactory); err != nil {
		return nil, err
	}

	// We must use binlog full row image.
	if err := b.canal.CheckBinlogRowImage("FULL"); err != nil {
		return nil, err
	}

	if err := b.newUpstream(cfg); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Bridge) newStateSaver(cfg *config.Config) error {
	fs, err := newFileSaver(cfg.App.DataFile, cfg.Replication.GTIDMode)
	if err != nil {
		return err
	}

	_, err = fs.load()
	if err != nil {
		return err
	}

	b.stateSaver = fs

	return nil
}

func (b *Bridge) newRules(cfg *config.Config, ehFactory EventHandlerFactory) error {
	db := cfg.Replication.SourceOpts.Database
	pluginDir := cfg.App.PluginDir

	rules := make(map[string]*mymy.Rule, len(cfg.Replication.Rules))
	for _, rule := range cfg.Replication.Rules {
		pluginCfg := rule.Upstream.Plugin
		uh, err := ehFactory.New(pluginCfg.Name, pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("create handler error: plugin dir: %s, name: %s, err: %w", pluginDir, pluginCfg.Name, err)
		}

		table := rule.Source.Table
		tableInfo, err := b.canal.GetTable(db, table)
		if err != nil {
			return err
		}

		pks := newColumnsFromPKs(tableInfo)
		if len(pks) == 0 {
			return fmt.Errorf("no primary keys found, schema: %s, table: %s", db, table)
		}

		cols := newColumnsFromNonPKs(tableInfo)

		rule := &mymy.Rule{
			Source: mymy.SourceInfo{
				Schema: db,
				Table:  table,
				PKs:    pks,
				Cols:   cols,
			},
			Handler: uh,
		}

		key := mymy.RuleKey(db, table)
		rules[key] = rule
	}

	b.rules = rules

	return nil
}

func (b *Bridge) updateRule(schema, table string) (*mymy.Rule, error) {
	rule, ok := b.rules[mymy.RuleKey(schema, table)]
	if !ok {
		return nil, ErrRuleNotExist
	}

	tableInfo, err := b.canal.GetTable(schema, table)
	if err != nil {
		return nil, err
	}

	rule.Source.PKs = newColumnsFromPKs(tableInfo)
	rule.Source.Cols = newColumnsFromNonPKs(tableInfo)

	return rule, nil
}

func (b *Bridge) newCanal(cfg *config.Config) error {
	canalCfg := canal.NewDefaultConfig()

	myCfg := cfg.Replication.SourceOpts
	if cfg.Replication.ServerID != nil {
		canalCfg.ServerID = *cfg.Replication.ServerID
	}
	canalCfg.Addr = myCfg.Addr
	canalCfg.User = myCfg.User
	canalCfg.Password = myCfg.Password
	canalCfg.Charset = myCfg.Charset
	canalCfg.Flavor = mysql.MySQLFlavor
	canalCfg.SemiSyncEnabled = false

	canalCfg.Dump.ExecutionPath = myCfg.Dump.ExecPath
	canalCfg.Dump.DiscardErr = false
	canalCfg.Dump.SkipMasterData = myCfg.Dump.SkipMasterData
	canalCfg.Dump.ExtraOptions = myCfg.Dump.ExtraOptions
	canalCfg.Dump.TableDB = myCfg.Database
	canalCfg.Dump.Tables = make([]string, 0, len(cfg.Replication.Rules))
	for _, mapping := range cfg.Replication.Rules {
		canalCfg.Dump.Tables = append(canalCfg.Dump.Tables, mapping.Source.Table)
	}

	syncOnly := make([]string, 0, len(cfg.Replication.Rules))
	for _, mapping := range cfg.Replication.Rules {
		regex := fmt.Sprintf("%s\\.%s", myCfg.Database, mapping.Source.Table)
		syncOnly = append(syncOnly, regex)
	}
	canalCfg.IncludeTableRegex = syncOnly

	cn, err := canal.NewCanal(canalCfg)
	if err != nil {
		return err
	}

	eH := newEventHandler(b, cfg.Replication.GTIDMode)
	cn.SetEventHandler(eH)

	b.canal = cn
	b.syncRulesAndCanalDump()

	return nil
}

func (b *Bridge) syncRulesAndCanalDump() {
	var db string
	dbs := map[string]struct{}{}
	tables := make([]string, 0, len(b.rules))
	for _, rule := range b.rules {
		db = rule.Source.Schema
		dbs[db] = struct{}{}
		tables = append(tables, rule.Source.Table)
	}

	if len(dbs) == 1 {
		b.canal.AddDumpTables(db, tables...)
	} else {
		keys := make([]string, 0, len(dbs))
		for key := range dbs {
			keys = append(keys, key)
		}

		b.canal.AddDumpDatabases(keys...)
	}
}

func (b *Bridge) newUpstream(cfg *config.Config) error {
	opts := &cfg.Replication.UpstreamOpts
	u, err := client.New(&client.Config{
		Addr:           opts.Addr,
		User:           opts.User,
		Password:       opts.Password,
		Database:       opts.Database,
		Charset:        opts.Charset,
		MaxRetries:     opts.MaxRetries,
		MaxOpenConns:   opts.MaxOpenConns,
		MaxIdleConns:   opts.MaxIdleConns,
		ConnectTimeout: opts.ConnectTimeout,
		WriteTimeout:   opts.WriteTimeout,
	})
	if err != nil {
		return err
	}
	b.upstream = u

	return nil
}

// Run syncs the data from MySQL and inserts to another MySQL
// until closed or meets errors.
func (b *Bridge) Run() error {
	defer b.setRunning(false)

	go b.runBackgroundJobs()

	errCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := b.syncLoop()
		if err != nil {
			errCh <- fmt.Errorf("sync loop error: %w", err)
			b.cancel()
		}
	}()

	b.setDumping(true)
	go func() {
		<-b.canal.WaitDumpDone()
		b.setDumping(false)
		b.setRunning(true)
	}()

	var err error
	pos := b.stateSaver.position()
	switch p := pos.(type) {
	case *gtidSet:
		err = b.canal.StartFromGTID(p.pos)
	case *binlogPos:
		err = b.canal.RunFrom(p.pos)
	default:
		err = errors.New("unsupported master position: expected GTID set or binlog file position")
	}

	if err != nil {
		errCh <- err
	}

	b.cancel()
	wg.Wait()
	close(errCh)

	var multi error
	for err := range errCh {
		multi = multierr.Append(multi, err)
	}

	return multi
}

func (b *Bridge) syncLoop() error {
	for {
		select {
		case got := <-b.syncCh:
			switch v := got.(type) {
			case *savePos:
				err := b.stateSaver.save(v.pos, v.force)
				if err != nil {
					return err
				}
			case batch:
				err := b.doBatch(v)
				if err != nil {
					return err
				}
			}
			b.syncedAt.Store(time.Now().Unix())
		case <-b.ctx.Done():
			return nil
		}
	}
}

func (b *Bridge) doBatch(queries batch) error {
	for _, query := range queries {
		q, args, err := query.SQL()
		if err != nil {
			b.logger.Err(err).
				Str("query", fmt.Sprintf("%+v", query)).
				Msg("could not convert to SQL statement")
		}

		_, err = b.upstream.Exec(context.Background(), q, args...)
		if err != nil {
			b.logger.Err(err).
				Str("query", q).
				Str("args", fmt.Sprintf("%+v", args)).
				Msg("could not exec SQL query")

			return err
		}
	}

	return nil
}

func (b *Bridge) Close() error {
	var err error
	b.closeOnce.Do(func() {
		b.canal.Close()
		b.cancel()

		err = multierr.Combine(
			b.stateSaver.close(),
			b.upstream.Close(),
		)
	})

	return err
}

func (b *Bridge) Delay() uint32 {
	return b.canal.GetDelay()
}

func (b *Bridge) setRunning(v bool) {
	b.running.Store(v)
	b.setDumping(false)

	if v {
		metrics.SetReplicationState(metrics.StateRunning)
	} else {
		metrics.SetReplicationState(metrics.StateStopped)
	}
}

func (b *Bridge) Running() bool {
	return b.running.Load()
}

func (b *Bridge) setDumping(v bool) {
	b.dumping.Store(v)
	if v {
		metrics.SetReplicationState(metrics.StateDumping)
	}
}

func (b *Bridge) Dumping() bool {
	return b.dumping.Load()
}

func (b *Bridge) runBackgroundJobs() {
	go func() {
		for range time.Tick(1 * time.Second) {
			metrics.SetSecondsBehindMaster(b.Delay())
		}
	}()

	go func() {
		for range time.Tick(1 * time.Second) {
			syncedAt := b.syncedAt.Load()
			if syncedAt > 0 {
				now := time.Now().Unix()
				metrics.SetSyncedSecondsAgo(now - syncedAt)
			}
		}
	}()
}
