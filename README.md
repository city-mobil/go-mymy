# MyMy

It is a service to replicate data from MySQL into MySQL automatically. MyMy behaves like a man in the middle and allows
to mutate and filter data.

It uses mysqldump to fetch the origin data at first, then syncs data incrementally with binlog.

MyMy build on top of plugins: you can use common plugins or write your own. Only plugins decide how to replicate the
data.

## Use cases

* You have a large source table and would like to replicate only some columns to another database.
* You need to mutate the data before passing it to the upstream database.
* You need to split the data flow from a source table to multiple tables.

In other cases strongly consider using the native MySQL replication.

## Requirements

* MySQL version >= 5.7, MariaDB is not supported right now.
* Binlog format must be set to ROW.
* Binlog row image must be full for MySQL. You may lost some field data if you update PK data in MySQL with minimal or
  noblob binlog row image.
* `mysqldump` must exist in the same node with replicator. If not, replicator will try to sync binlog only.

### MySQL

Create or use exist user with the replication grants on source database:

```mysql
GRANT PROCESS, RELOAD, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%';
FLUSH PRIVILEGES;
```

MyMy supports two dump options:

1. Mutate and import the initial data row by row. It is safe to use but it is really slow.
2. Use the `LOAD DATA LOCAL INFILE` statement which faster significantly but requires enabling the option 
   `local_infile` on the database side. Read more [here](https://dev.mysql.com/doc/refman/8.0/en/load-data.html).

To use the second approach set option `load_in_file_enabled` to true.

## API

Replicator exposes several debug endpoints:

* `/metrics` - runtime and app metrics in Prometheus format,
* `/health` - health check,
* `/about` - shows app version and build information.

Health check returns status `503 Service Unavailable` if replicator is not running, dumping data or replication lag
greater than `app.health.seconds_behind_master` config value.

## Writing plugin

Implement an interface `mymy.EventHandler` and define a constructor `NewEventHandler`:

```go
package main

import "github.com/city-mobil/go-mymy/pkg/mymy"

type MyEventHandler struct{}

func (eH *MyEventHandler) OnTableChanged(info mymy.SourceInfo) error {
	// Do something after scheme alter. 
	panic("implement me")
}

func (eH *MyEventHandler) OnRows(e *mymy.RowsEvent) ([]*mymy.Query, error) {
	// Do something with the event.
	panic("implement me")
}

func NewEventHandler(_ string) (mymy.EventHandler, error) {
	return &MyEventHandler{}, nil
}
```

The handler constructor might get a path to its configuration file.

## How to build the replicator and custom plugins

You must build the package and your plugins on the same machine unless you get an error like:

```
plugin was built with a different version of package ...
```

Clone the `go-mymy` repository. Add the next line to your `go.mod` file in the plugin directory:

```
replace github.com/city-mobil/go-mymy v1.1.5 => ../go-mymy
```

Where `../go-mymy` is the relative path to the cloned replicator repository.

Build the plugin:

```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o bin/my_plugin.so my_plugin/main.go
```

To build the `deb` package with the replicator install [goreleaser](https://goreleaser.com/install/) and run:

```bash
cd go-mymy
goreleaser --skip-publish --rm-dist
```

The package will be saved in the `dist` directory.

## Frequently Asked Questions

#### Got the error "canal dump mysql err: exit status 2"  

Try to increase the MySQL option `wait_timeout`:

```mysql
SET @@GLOBAL.wait_timeout = 240;
```
