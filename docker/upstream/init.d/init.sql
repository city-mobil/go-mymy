GRANT ALL PRIVILEGES ON *.* TO 'repl'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;

CREATE TABLE clients
(
    id    bigint unsigned auto_increment primary key,
    name  varchar(50) default '' not null,
    email varchar(254)           not null
) charset = utf8;
