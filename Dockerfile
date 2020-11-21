FROM golang:1.14.11 as builder

RUN apt-get update && apt-get install -y mariadb-client

WORKDIR /replicator
COPY go.mod go.sum ./

RUN go mod download

COPY . ./
RUN make build


FROM golang:1.14.11

WORKDIR /usr/bin
COPY --from=builder /replicator/bin/mymy ./
COPY --from=builder /usr/bin/mysqldump /usr/bin/mysqldump

RUN chmod +x ./mymy
COPY config/mymy.conf.yml /etc/mymy/conf.yml

ENTRYPOINT ["/usr/bin/mymy"]
CMD ["-config", "/etc/mymy/conf.yml"]