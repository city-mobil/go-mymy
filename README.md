# MyMy

It is a service to replicate data from MySQL into MySQL automatically. 
MyMy behaves like a man in the middle and allows to mutate and filter data.

It uses mysqldump to fetch the origin data at first, then syncs data incrementally with binlog.

MyMy build on top of plugins: you can use ready plugins or write your own. 
Only plugins decides how to replicate the data. 

## Use cases

* You have a large source table and would like to replicate only some columns to another database.
* You need to mutate the data before passing it to the upstream database.
* You need to split the data flow from a source table to multiple tables.

In other cases strongly consider using the native MySQL replication.

## Requirements

* MySQL version >= 5.7, MariaDB is not supported right now.
* Binlog format must be set to ROW.
* Binlog row image must be full for MySQL. You may lost some field data if you update PK data in MySQL with minimal or noblob binlog row image.
* `mysqldump` must exist in the same node with mysql-tarantool-replicator. If not, replicator will try to sync binlog only.

### MySQL

Create or use exist user with the replication grants on source database:

```mysql
GRANT PROCESS, RELOAD, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%';
FLUSH PRIVILEGES;
```

## Writing plugin

Implement an interface `mymy.EventHandler` and define a constructor `NewEventHandler`:

```go
package main

import "github.com/city-mobil/go-mymy/pkg/mymy"

type MyEventHandler struct {}

func (eH *MyEventHandler) OnRows(e *mymy.RowsEvent) ([]*mymy.Query, error) {
    // Do something with the event.
    panic("implement me")
}

func NewEventHandler(_ string) (mymy.EventHandler, error) {
	return &MyEventHandler{}, nil
}
```

The handler constructor might get a path to its configuration file.