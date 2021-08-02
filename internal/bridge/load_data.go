package bridge

import (
	"fmt"
	"os"

	"github.com/city-mobil/go-mymy/internal/config"
	"github.com/city-mobil/go-mymy/pkg/mymy"
)

type file struct {
	localPath  string
	dockerPath string
	cntRows    int
	descriptor *os.File
}

func newFile(table, localPathPrefix, dockerPathPrefix string) (*file, error) {
	lPath := fmt.Sprintf("%s%s", completPrefix(localPathPrefix), table)
	dPath := fmt.Sprintf("%s%s", completPrefix(dockerPathPrefix), table)
	f, err := os.Create(lPath)
	if err != nil {
		return nil, err
	}

	return &file{
		localPath:  lPath,
		dockerPath: dPath,
		descriptor: f,
	}, nil
}

type loader struct {
	data             map[string]*file
	rowsLimit        int
	localPathPrefix  string
	dockerPathPrefix string
}

func newLoader(cfg *config.Config) *loader {
	return &loader{
		data:             make(map[string]*file),
		rowsLimit:        cfg.Replication.SourceOpts.Dump.DumpSize,
		localPathPrefix:  cfg.Replication.SourceOpts.Dump.LocalPathDumpFile,
		dockerPathPrefix: cfg.Replication.SourceOpts.Dump.DockerPathDumpFile,
	}
}

func (l *loader) placeReq(queries batch, dbName string) error {
	for _, query := range queries {
		t := fmt.Sprintf("%s.%s", dbName, query.Table)
		if _, ok := l.data[t]; !ok {
			f, err := newFile(t, l.localPathPrefix, l.dockerPathPrefix)
			if err != nil {
				return err
			}

			l.data[t] = f
		}

		err := l.writeRowInFile(query, t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) writeRowInFile(query *mymy.Query, table string) error {
	args := make([]interface{}, len(query.Values))
	for i, arg := range query.Values {
		args[i] = arg.Value
	}

	err := l.data[table].writeRow(args)
	if err != nil {
		return err
	}

	l.data[table].cntRows++

	return nil
}

func (f *file) writeRow(args []interface{}) error {
	_, err := f.descriptor.WriteString(joinInterfaces(",", args))

	return err
}

func joinInterfaces(delim string, args []interface{}) (str string) {
	if len(args) == 0 {
		return str
	}

	str = fmt.Sprintf(`"%v"`, args[0])
	for i := 1; i < len(args); i++ {
		str += fmt.Sprintf(`%s"%v"`, delim, args[i])
	}

	return str + "\n"
}

func (l *loader) closeFiles() {
	for _, f := range l.data {
		f.descriptor.Close()
	}
}

func completPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	return prefix + "employee_data_"
}
