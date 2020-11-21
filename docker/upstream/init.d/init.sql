GRANT ALL PRIVILEGES ON *.* TO 'repl'@'%';

CREATE TABLE clients
(
    id    bigint unsigned auto_increment primary key,
    name  varchar(50) default '' not null,
    email varchar(254)           not null
) charset = utf8;

CREATE TABLE logins
(
    username varchar(16)  not null,
    ip       varchar(16)  not null,
    date     int unsigned not null,
    attempts int unsigned default 0,

    PRIMARY KEY (username, ip, date)
) charset = utf8;