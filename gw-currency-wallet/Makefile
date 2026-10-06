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
	docker compose -f $(DOCKER_CTX)/docker-compose.yml up -d wallet-db gw-exchanger kafka kafka-init $(APP_NAME)

docker-down: ## Остановить сервис
	docker compose -f $(DOCKER_CTX)/docker-compose.yml down

logs: ## Логи сервиса
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f $(APP_NAME)

logs-kafka: ## Логи Kafka
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f kafka

logs-db: ## Логи wallet-db
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f wallet-db

ps: ## Статус контейнеров
	docker compose -f $(DOCKER_CTX)/docker-compose.yml ps

migrate: ## Применить миграцию (init.sql) к wallet-db
	docker compose -f $(DOCKER_CTX)/docker-compose.yml cp \
		./internal/migrations/001_init.sql wallet-db:/tmp/init.sql
	docker compose -f $(DOCKER_CTX)/docker-compose.yml exec -T wallet-db \
		psql -U postgres -d wallet -f /tmp/init.sql