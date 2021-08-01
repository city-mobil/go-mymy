package bridge

import (
	"fmt"
	"github.com/city-mobil/go-mymy/pkg/mymy"
	"github.com/rs/zerolog"
	"os"
)

const (
	localPathPrefix  = "bin/tmp/employee_data_"
	dockerPathPrefix = "/opt/dump/employee_data_"
)

type file struct {
	localPath  string
	dockerPath string
	cntRows    int
	descriptor *os.File
}

func newFile(table string) (*file, error) {
	lPath := fmt.Sprintf("%s%s", localPathPrefix, table)
	dPath := fmt.Sprintf("%s%s", dockerPathPrefix, table)
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
	data   map[string]*file
	logger zerolog.Logger
}

func newLoader(logger zerolog.Logger) *loader {
	return &loader{
		data:   make(map[string]*file),
		logger: logger,
	}
}

func (l *loader) placeReq(queries batch, dbName string) error {
	for _, query := range queries {
		t := fmt.Sprintf("%s.%s", dbName, query.Table)
		if _, ok := l.data[t]; !ok {
			fmt.Println(2, t)
			f, err := newFile(t)
			fmt.Println(l.data)
			if err != nil {
				return err
			}

			l.data[t] = f
		}

		l.writeRowInFile(query, t)
	}

	return nil
}

func (l *loader) writeRowInFile(query *mymy.Query, table string) {
	_, args, err := query.SQL()
	if err != nil {
		l.logger.Err(err).
			Str("query", fmt.Sprintf("%+v", query)).
			Msg("could not convert to SQL statement")
	}

	err = l.data[table].writeRow(args)
	if err != nil {
		l.logger.Err(err).
			Str("query", fmt.Sprintf("%+v", query)).
			Str("table", table).
			Msg("it was not possible to write to the csv file")
	} else {
		l.data[table].cntRows++
	}
}

func (f *file) writeRow(args []interface{}) error {
	_, err := fmt.Fprintf(f.descriptor, "%s\n", joinInterfaces(",", args))
	return err
}

func joinInterfaces(delim string, args []interface{}) (str string) {
	if len(args) == 0 {
		return str
	}

	str = fmt.Sprintf("%v", args[0])
	for i := 1; i < len(args); i++ {
		str += fmt.Sprintf("%s%v", delim, args[i])
	}

	return str
}

//load data infile '/opt/dump/employee_data_town.clients' into table town.clients  fields terminated by ','  lines terminated by '\n' - right;
