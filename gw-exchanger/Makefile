APP_NAME := gw-exchanger
MAIN_PKG := ./cmd

.PHONY: build run test test-unit test-integration tidy fmt vet docker-build docker-up docker-down migrate

build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME) $(MAIN_PKG)

build-local:
	go build -o bin/$(APP_NAME) $(MAIN_PKG)

run: build-local #для запуска этой командой нужна локально развернутая БД
	./bin/$(APP_NAME)

test:
	go test ./... -count=1 -race

test-unit:
	go test ./internal/server/... ./pkg/... -count=1 -race

test-integration:
	TEST_DSN="postgres://postgres:postgres@localhost:5432/exchanger?sslmode=disable" \
		go test ./internal/storages/... -count=1 -v

tidy:
	go mod tidy

fmt:
	gofmt -s -w .

vet:
	go vet ./...

migrate:
	psql "postgres://postgres:postgres@localhost:5432/exchanger?sslmode=disable" \
		-f internal/migrations/001_init.sql

docker-build:
	docker build -t $(APP_NAME):latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down