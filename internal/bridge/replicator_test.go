package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/city-mobil/go-mymy/internal/client"
	"github.com/city-mobil/go-mymy/internal/config"
	mymy_mock "github.com/city-mobil/go-mymy/mock"
	"github.com/city-mobil/go-mymy/pkg/mymy"
)

type baseFactory struct {
	table string
	skip  []string
}

func (f *baseFactory) New(_, _ string) (mymy.EventHandler, error) {
	h := mymy.NewBaseEventHandler(f.table)
	h.Skip(f.skip)

	return h, nil
}

type bridgeSuite struct {
	suite.Suite

	bridge   *Bridge
	source   *client.SQLClient
	upstream *client.SQLClient
	logger   zerolog.Logger
	cfg      *config.Config
}

func (s *bridgeSuite) init(cfg *config.Config, ehFactory EventHandlerFactory) {
	b, err := New(cfg, ehFactory, s.logger)
	require.NoError(s.T(), err)

	s.bridge = b
}

func TestReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in a short mode.")
	}

	cfgPath, err := filepath.Abs("testdata/cfg.yml")
	require.NoError(t, err)

	cfg, err := config.ReadFromFile(cfgPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	logger := zerolog.New(zerolog.NewConsoleWriter())

	sOpts := cfg.Replication.SourceOpts
	sClient, err := client.New(&client.Config{
		Addr:       sOpts.Addr,
		User:       sOpts.User,
		Password:   sOpts.Password,
		Database:   sOpts.Database,
		Charset:    sOpts.Charset,
		MaxRetries: 2,
	})
	require.NoError(t, err)

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
	require.NoError(t, err)

	suite.Run(t, &bridgeSuite{
		source:   sClient,
		upstream: uClient,
		logger:   logger,
		cfg:      cfg,
	})
}

func (s *bridgeSuite) hasSyncedPos() bool {
	syncedGTID := s.bridge.canal.SyncedGTIDSet()
	savedPos := s.bridge.stateSaver.position()

	return savedPos.equal(newGTIDSet(syncedGTID))
}

func (s *bridgeSuite) hasSyncedData(rows int) bool {
	res := s.upstream.QueryRow(context.Background(), "SELECT COUNT(*) FROM town.clients")

	var cnt int
	err := res.Scan(&cnt)
	if assert.NoError(s.T(), err) {
		return cnt == rows
	}

	return false
}

func (s *bridgeSuite) AfterTest(_, _ string) {
	t := s.T()

	if s.bridge != nil {
		err := s.bridge.Close()
		assert.NoError(t, err)
	}

	_, err := s.source.Exec(context.Background(), "TRUNCATE city.users")
	assert.NoError(t, err)

	_, err = s.upstream.Exec(context.Background(), "TRUNCATE town.clients")
	assert.NoError(t, err)

	dataDir := path.Dir(s.cfg.App.DataFile)
	err = os.RemoveAll(dataDir)
	assert.NoError(t, err)
}

func (s *bridgeSuite) TestNewBridge() {
	factory := &baseFactory{
		table: "clients",
	}

	b, err := New(s.cfg, factory, s.logger)
	require.NoError(s.T(), err)

	s.bridge = b
}

func (s *bridgeSuite) TestDump() {
	t := s.T()
	dumpPath := "/usr/bin/mysqldump"
	if !assert.FileExists(t, dumpPath) {
		t.Skip("test requires mysqldump utility")
	}

	factory := &baseFactory{
		table: "clients",
		skip:  []string{"username", "password"},
	}

	cfg := *s.cfg
	cfg.Replication.SourceOpts.Dump.ExecPath = dumpPath
	s.init(&cfg, factory)

	rows := 200

	// Prepare initial data.
	for i := 1; i <= rows; i++ {
		_, err := s.source.Exec(context.Background(), "INSERT INTO city.users (id, username, password, name, email) VALUES (?, ?, ?, ?, ?)", i, "bob", "12345", "Bob", "bob@email.com")
		require.NoError(t, err)
	}

	go func() {
		err := s.bridge.Run()
		assert.NoError(t, err)
	}()

	<-s.bridge.canal.WaitDumpDone()

	require.Eventually(t, func() bool {
		return s.hasSyncedData(rows)
	}, 10*time.Second, 50*time.Millisecond)

	err := s.bridge.Close()
	assert.NoError(t, err)
}

func (s *bridgeSuite) TestReplication() {
	t := s.T()

	factory := &baseFactory{
		table: "clients",
		skip:  []string{"username", "password"},
	}
	s.init(s.cfg, factory)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	go func() {
		err := s.bridge.Run()
		assert.NoError(t, err)
		cancel()
	}()

	<-s.bridge.canal.WaitDumpDone()

	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()

	rows := 200
	bobs := 0

tank:
	for bobs < rows {
		select {
		case <-tick.C:
			_, err := s.source.Exec(context.Background(), "INSERT INTO city.users (username, password, name, email) VALUES (?, ?, ?, ?)", "bob", "12345", "Bob", "bob@email.com")
			if assert.NoError(t, err) {
				bobs++
			}
			_, err = s.source.Exec(context.Background(), "INSERT INTO city.users (username, password, name, email) VALUES (?, ?, ?, ?)", "alice", "qwerty", "Alice", "alice@email.com")
			assert.NoError(t, err)
		case <-ctx.Done():
			break tank
		}
	}

	_, err := s.source.Exec(context.Background(), "DELETE FROM city.users where username=?", "alice")
	require.NoError(t, err)

	_, err = s.source.Exec(context.Background(), "UPDATE city.users SET password = ?, email = ? where username = ?", "11111", "boby@gmail.com", "bob")
	require.NoError(t, err)

	err = s.bridge.canal.CatchMasterPos(500 * time.Millisecond)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return s.hasSyncedData(bobs)
	}, 1*time.Second, 50*time.Millisecond)

	assert.Eventually(t,
		s.hasSyncedPos,
		500*time.Millisecond,
		50*time.Millisecond,
		"bridge: %s, master: %s", s.bridge.stateSaver.position(), s.bridge.canal.SyncedGTIDSet(),
	)

	err = s.bridge.Close()
	assert.NoError(t, err)
}

func (s *bridgeSuite) TestReconnect() {
	t := s.T()

	factory := &baseFactory{
		table: "clients",
		skip:  []string{"username", "password"},
	}

	for i := 1; i < 5; i++ {
		s.init(s.cfg, factory)

		go func() {
			err := s.bridge.Run()
			assert.NoError(t, err)
		}()

		name := fmt.Sprintf("dead_cow_%d", i)
		_, err := s.source.Exec(context.Background(), "INSERT INTO city.users (username, password, name, email) VALUES (?, ?, ?, ?)", name, "12345", name, "robot@email.com")
		require.NoError(t, err)

		err = s.bridge.canal.CatchMasterPos(500 * time.Millisecond)
		require.NoError(t, err)

		wantRows := i
		require.Eventually(t, func() bool {
			return s.hasSyncedData(wantRows)
		}, 500*time.Millisecond, 50*time.Millisecond)

		err = s.bridge.Close()
		assert.NoError(t, err)
	}
}

func (s *bridgeSuite) TestRenameColumn() {
	t := s.T()

	factory := &baseFactory{
		table: "clients",
		skip:  []string{"username", "password", "new_name"},
	}

	s.init(s.cfg, factory)

	go func() {
		err := s.bridge.Run()
		assert.NoError(t, err)
	}()

	<-s.bridge.canal.WaitDumpDone()

	_, err := s.source.Exec(context.Background(), "INSERT INTO city.users (username, password, name, email) VALUES (?, ?, ?, ?)", "bob", "12345", "Bob", "bob@email.com")
	require.NoError(t, err)

	_, err = s.source.Exec(context.Background(), "ALTER TABLE city.users CHANGE `name` `new_name` varchar(50)")
	require.NoError(t, err)

	defer func() {
		_, err = s.source.Exec(context.Background(), "ALTER TABLE city.users CHANGE `new_name` `name` varchar(50)")
		require.NoError(t, err)
	}()

	_, err = s.source.Exec(context.Background(), "INSERT INTO city.users (id, username, password, new_name, email) VALUES (?, ?, ?, ?, ?)", 2, "alice", "123", "Alice", "alice@email.com")
	require.NoError(t, err)

	err = s.bridge.canal.CatchMasterPos(500 * time.Millisecond)
	require.NoError(t, err)

	wantRows := 2
	require.Eventually(t, func() bool {
		return s.hasSyncedData(wantRows)
	}, 500*time.Millisecond, 50*time.Millisecond)

	row := s.upstream.QueryRow(context.Background(), "SELECT name, email FROM town.clients WHERE id=?", 2)
	require.NotNil(t, row)
	var name, email string
	err = row.Scan(&name, &email)
	require.NoError(t, err)
	assert.Equal(t, "", name)
	assert.Equal(t, "alice@email.com", email)

	err = s.bridge.Close()
	assert.NoError(t, err)
}

type mockFactory struct {
	handler mymy.EventHandler
}

func (f *mockFactory) New(_, _ string) (mymy.EventHandler, error) {
	return f.handler, nil
}

func (s *bridgeSuite) TestAlterHandler() {
	t := s.T()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := mymy_mock.NewMockEventHandler(ctrl)
	factory := &mockFactory{
		handler: handler,
	}
	s.init(s.cfg, factory)

	var wg sync.WaitGroup
	wg.Add(1)

	handler.EXPECT().OnTableChanged(gomock.Any()).DoAndReturn(func(got mymy.SourceInfo) error {
		defer wg.Done()

		found := false
		for _, col := range got.Cols {
			if col.Name == "new_name" {
				found = true

				break
			}
		}
		assert.True(t, found, "altered column not found")

		return nil
	}).Times(1)

	go func() {
		err := s.bridge.Run()
		assert.NoError(t, err)
	}()

	<-s.bridge.canal.WaitDumpDone()

	_, err := s.source.Exec(context.Background(), "ALTER TABLE city.users CHANGE `name` `new_name` varchar(50)")
	require.NoError(t, err)

	defer func() {
		_, err = s.source.Exec(context.Background(), "ALTER TABLE city.users CHANGE `new_name` `name` varchar(50)")
		require.NoError(t, err)
	}()

	wg.Wait()

	err = s.bridge.Close()
	assert.NoError(t, err)
}

func (s *bridgeSuite) TestHandlerReturnsError() {
	t := s.T()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := mymy_mock.NewMockEventHandler(ctrl)
	factory := &mockFactory{
		handler: handler,
	}
	s.init(s.cfg, factory)

	handler.EXPECT().OnRows(gomock.Any()).Return(nil, errors.New("fatal")).Times(1)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := s.bridge.Run()
		assert.Error(t, err)
	}()

	<-s.bridge.canal.WaitDumpDone()

	_, err := s.source.Exec(context.Background(), "INSERT INTO city.users (username, password, name, email) VALUES (?, ?, ?, ?)", "bob", "12345", "Bob", "bob@email.com")
	require.NoError(t, err)

	wg.Wait()

	err = s.bridge.Close()
	assert.NoError(t, err)
}

type errFactory struct {
}

func (f *errFactory) New(_, _ string) (mymy.EventHandler, error) {
	return nil, errors.New("fatal")
}

func (s *bridgeSuite) TestFactoryReturnsError() {
	t := s.T()

	b, err := New(s.cfg, &errFactory{}, s.logger)
	assert.Error(t, err)
	assert.Nil(t, b)
}
