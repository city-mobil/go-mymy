package bridge

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/city-mobil/go-mymy/internal/client"
	"github.com/city-mobil/go-mymy/pkg/mymy"
	"github.com/go-sql-driver/mysql"
)

const argSeparator = ","

type inFileLoader struct {
	data           map[string]batch
	database       string
	upstream       *client.SQLClient
	flushThreshold int
	argEnclose     string
}

func newInFileLoader(database string, upstream *client.SQLClient, flushThreshold int, argEnclose string) *inFileLoader {
	return &inFileLoader{
		data:           make(map[string]batch),
		database:       database,
		upstream:       upstream,
		flushThreshold: flushThreshold,
		argEnclose:     argEnclose,
	}
}

func (loader *inFileLoader) append(queries batch) error {
	for _, query := range queries {
		key := loader.buildKey(loader.database, query.Table)
		b, ok := loader.data[key]
		if !ok {
			b = make(batch, 0)
		}

		b = append(b, query)
		loader.data[key] = b
	}

	for key, b := range loader.data {
		if len(b) >= loader.flushThreshold {
			err := loader.flush(key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (loader *inFileLoader) flush(key string) error {
	b := loader.data[key]
	if len(b) == 0 {
		return nil
	}
	defer func() {
		loader.data[key] = make(batch, 0)
	}()

	f, err := ioutil.TempFile("", "mymy")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	for _, query := range b {
		str := loader.buildDumpRow(query)
		if str == "" {
			continue
		}

		_, err = f.WriteString(str)
		if err != nil {
			return err
		}
	}

	err = f.Close()
	if err != nil {
		return err
	}

	mysql.RegisterLocalFile(f.Name())
	defer mysql.DeregisterLocalFile(f.Name())

	q := loader.buildDumpQuery(f.Name(), key)
	_, err = loader.upstream.Exec(context.Background(), q)

	return err
}

func (loader *inFileLoader) flushAll() error {
	for key := range loader.data {
		err := loader.flush(key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (loader *inFileLoader) buildDumpQuery(filepath, table string) string {
	return fmt.Sprintf(
		`LOAD DATA LOCAL INFILE '%s' INTO TABLE %s FIELDS TERMINATED BY '%s' ENCLOSED BY '%s' LINES TERMINATED BY '\n'`,
		filepath, table, argSeparator, loader.argEnclose,
	)
}

func (loader *inFileLoader) buildDumpRow(query *mymy.Query) string {
	if len(query.Values) == 0 {
		return ""
	}

	args := make([]interface{}, len(query.Values))
	for i, arg := range query.Values {
		args[i] = arg.Value
	}

	argEnclose := loader.argEnclose
	var sb strings.Builder
	sb.WriteString(argEnclose)
	sb.WriteString(fmt.Sprintf("%v", args[0]))
	sb.WriteString(argEnclose)
	for _, arg := range args[1:] {
		sb.WriteString(argSeparator)
		sb.WriteString(argEnclose)
		sb.WriteString(fmt.Sprintf("%v", arg))
		sb.WriteString(argEnclose)
	}
	sb.WriteString("\n")

	return sb.String()
}

func (loader *inFileLoader) buildKey(db, table string) string {
	var sb strings.Builder
	sb.Grow(len(db) + len(table) + 1)
	sb.WriteString(db)
	sb.WriteRune('.')
	sb.WriteString(table)

	return sb.String()
}
