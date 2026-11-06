GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)

.PHONY: init
# init env
init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

.PHONY: config
# generate internal proto
config:
	buf generate --template buf.gen.config.yaml

.PHONY: api
# generate api proto
api:
	buf generate --template buf.gen.yaml

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api
	make config
	make generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

.PHONY: compose-up
# start the full mall stack (MySQL, Redis, RabbitMQ, Elasticsearch, app)
compose-up:
	docker compose up -d --build

.PHONY: compose-down
# stop the stack and remove its volumes
compose-down:
	docker compose down -v

.PHONY: compose-logs
# follow application logs
compose-logs:
	docker compose logs -f app

.PHONY: compose-ps
# show stack health
compose-ps:
	docker compose ps

.PHONY: seed
# load demo data after the app has auto-migrated the schema, then restart the
# app so the startup goods re-index (EnsureIndex) picks up the new rows
seed:
	docker compose exec -T mysql mysql -uroot -p$${MYSQL_ROOT_PASSWORD:-root} mall < scripts/seed.sql
	docker compose restart app

.PHONY: smoke
# run the end-to-end mall smoke test
smoke:
	bash scripts/smoke.sh
