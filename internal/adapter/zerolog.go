package adapter

import (
	"bytes"
	"strings"

	"github.com/rs/zerolog"
)

// ZeroLogHandler is a handler to redirect siddontang/go-log messages to zerolog.
type ZeroLogHandler struct {
	logger zerolog.Logger
}

func NewZeroLogHandler(logger zerolog.Logger) *ZeroLogHandler {
	return &ZeroLogHandler{
		logger: logger,
	}
}

func (h *ZeroLogHandler) Write(p []byte) (n int, err error) {
	level, msg := parseLevelAndMsg(p)
	h.logger.WithLevel(level).Msg(msg)

	return len(p), nil
}

func (h *ZeroLogHandler) Close() error {
	return nil
}

func parseLevelAndMsg(p []byte) (level zerolog.Level, msg string) {
	defLevel := zerolog.InfoLevel
	if len(p) == 0 || p[0] != '[' {
		return defLevel, string(p)
	}

	end := bytes.IndexByte(p[1:], ']')
	if end == -1 {
		return defLevel, string(p)
	}
	end++

	level, err := zerolog.ParseLevel(string(p[1:end]))
	if err != nil {
		return defLevel, string(p)
	}

	return level, strings.TrimSpace(string(p[end+1:]))
}
