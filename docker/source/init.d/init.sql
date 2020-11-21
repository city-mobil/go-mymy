GRANT PROCESS, RELOAD, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%';

CREATE TABLE users
(
    id       bigint unsigned auto_increment primary key,
    username varchar(16)            not null,
    password varchar(254)           not null,
    name     varchar(50) default '' not null,
    email    varchar(254)           not null
) charset = utf8;

CREATE TABLE logins
(
    username  varchar(16)  not null,
    ip        varchar(16)  not null,
    date      int unsigned not null,
    attempts  int unsigned   default 0,
    longitude float unsigned default 0,
    latitude  float unsigned default 0,

    PRIMARY KEY (username, ip, date)
) charset = utf8;