GRANT PROCESS, RELOAD, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%';

CREATE TABLE users
(
    id       bigint unsigned auto_increment primary key,
    username varchar(16)            not null,
    password varchar(254)           not null,
    name     varchar(50) default '' not null,
    email    varchar(254)           not null
) charset = utf8;
