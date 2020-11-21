package util

import (
	"os"
	"path"
)

var execPath = mustGetExecPath()

func mustGetExecPath() string {
	p, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return p
}

func AbsPath(src string) string {
	if path.IsAbs(src) {
		return src
	}

	return path.Join(path.Dir(execPath), src)
}
