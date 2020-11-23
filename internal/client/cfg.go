package client

import (
	"strings"
	"time"
)

type Config struct {
	Addr           string
	User           string
	Password       string
	Database       string
	Charset        string
	MaxRetries     int
	MaxOpenConns   int
	MaxIdleConns   int
	ConnectTimeout time.Duration
	WriteTimeout   time.Duration
}

func (c *Config) DSN() string {
	builder := strings.Builder{}
	builder.WriteString(c.User)
	builder.WriteByte(':')
	builder.WriteString(c.Password)
	builder.WriteString("@tcp(")
	builder.WriteString(c.Addr)
	builder.WriteString(")/")
	builder.WriteString(c.Database)
	builder.WriteString("?interpolateParams=true")
	if c.ConnectTimeout != 0 {
		builder.WriteString("&timeout=")
		builder.WriteString(c.ConnectTimeout.String())
	}
	if c.WriteTimeout != 0 {
		builder.WriteString("&writeTimeout=")
		builder.WriteString(c.WriteTimeout.String())
	}
	if c.Charset != "" {
		builder.WriteString("&charset=")
		builder.WriteString(c.Charset)
	}

	return builder.String()
}
