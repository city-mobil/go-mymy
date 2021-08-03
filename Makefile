BINARY=mymy
VERSION=`git describe --tags --dirty --always`
COMMIT=`git rev-parse HEAD`
BUILD_DATE=`date +%FT%T%z`
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

all: build

.PHONY: gen
gen:
	go generate

.PHONY: build
build: gen
	go build ${LDFLAGS} -o bin/${BINARY} cmd/mymy/main.go
	go build -buildmode=plugin -o bin/plugins/mymy_filter.so cmd/plugins/filter/main.go
	cp cmd/plugins/filter/cfg.yml bin/plugins/filter.plugin.yml
	go build -o bin/dump_benchmark cmd/dump_benchmark/main.go

.PHONY: lint
lint:
	golangci-lint run -v ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: run
run: build
	bin/${BINARY} -config=config/dev.conf.yml

.PHONY: run_dump_benchmark
run_dump_benchmark: build
	rm -rf bin/tmp/*
	rm -f bin/state.info
	bin/dump_benchmark -config=config/dump_benchmark.conf.yml

.PHONY: run_short_tests
run_short_tests:
	go test -count=1 -v -short ./...

.PHONY: run_tests
run_tests: env_up
	go test -count=1 -v -race -coverpkg=./internal/... -covermode=atomic -coverprofile=profile.cov ./...
	go tool cover -func=profile.cov
	go tool cover -html=profile.cov -o cover.html
	rm profile.cov

.PHONY: env_up
env_up:
	docker-compose up -d
	./docker/wait.sh
	docker-compose ps

.PHONY: env_down
env_down:
	docker-compose down -v --rmi local --remove-orphans