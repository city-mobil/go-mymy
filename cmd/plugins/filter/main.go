//nolint:unused,deadcode
package main

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/city-mobil/go-mymy/pkg/mymy"
)

type config struct {
	Table string   `yaml:"table"`
	Sync  []string `yaml:"sync,omitempty"`
	Skip  []string `yaml:"skip,omitempty"`
}

func readConfig(path string) (*config, error) {
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

	var cfg config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

type FilterEventHandler struct {
	def *mymy.BaseEventHandler
}

func NewEventHandler(cfgPath string) (mymy.EventHandler, error) {
	cfg, err := readConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	def := mymy.NewBaseEventHandler(cfg.Table)
	if cfg.Sync != nil {
		def.SyncOnly(cfg.Sync)
	}
	if cfg.Skip != nil {
		def.Skip(cfg.Skip)
	}

	return &FilterEventHandler{
		def: def,
	}, nil
}

func (eH *FilterEventHandler) OnTableChanged(info mymy.SourceInfo) error {
	return eH.def.OnTableChanged(info)
}

func (eH *FilterEventHandler) OnRows(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	return eH.def.OnRows(e)
}
