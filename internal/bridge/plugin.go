package bridge

import (
	"path"
	"plugin"

	"github.com/city-mobil/go-mymy/internal/util"
	"github.com/city-mobil/go-mymy/pkg/mymy"
)

func newUserHandler(searchDir, name, cfgPath string) (mymy.EventHandler, error) {
	pluginPath := util.AbsPath(path.Join(searchDir, name+".so"))
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, err
	}

	newHandler, err := p.Lookup("NewEventHandler")
	if err != nil {
		return nil, err
	}

	absCfgPath := util.AbsPath(cfgPath)

	return newHandler.(func(cfgPath string) (mymy.EventHandler, error))(absCfgPath)
}
