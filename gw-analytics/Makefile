tidy: ## go mod tidy
	go mod tidy

fmt: ## gofmt
	gofmt -s -w .

vet: ## go vet
	go vet ./...

swag: ## Сгенерировать swagger docs
	swag init -g cmd/main.go -o docs --parseDependency --parseInternal

docker-build: ## Пересобрать образ
	docker compose -f $(DOCKER_CTX)/docker-compose.yml build --no-cache $(APP_NAME)

docker-up: ## Поднять сервис (и зависимости)
	docker compose -f $(DOCKER_CTX)/docker-compose.yml up -d clickhouse kafka kafka-init $(APP_NAME)

docker-down: ## Остановить сервис
	docker compose -f $(DOCKER_CTX)/docker-compose.yml down	

logs: ## Логи сервиса
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f $(APP_NAME)

logs-kafka: ## Логи Kafka
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f kafka

logs-clickhouse: ## Логи ClickHouse
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f clickhouse

ps: ## Статус контейнеров
	docker compose -f $(DOCKER_CTX)/docker-compose.yml ps

migrate: ## Применить миграцию (init.sql) к ClickHouse
	docker compose -f $(DOCKER_CTX)/docker-compose.yml cp \
		./internal/migrations/001_init.sql clickhouse:/tmp/init.sql
	docker compose -f $(DOCKER_CTX)/docker-compose.yml exec clickhouse \
		clickhouse-client --multiquery --queries-file /tmp/init.sql