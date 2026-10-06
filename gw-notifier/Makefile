APP_NAME   := gw-notifier
DOCKER_CTX := ..

tidy: ## go mod tidy
	go mod tidy

fmt: ## gofmt
	gofmt -s -w .

vet: ## go vet
	go vet ./...

docker-build: ## Пересобрать образ
	docker compose -f $(DOCKER_CTX)/docker-compose.yml build --no-cache $(APP_NAME)

docker-up: ## Поднять сервис (и зависимости)
	docker compose -f $(DOCKER_CTX)/docker-compose.yml up -d mongo kafka kafka-init $(APP_NAME)

docker-down: ## Остановить сервис
	docker compose -f $(DOCKER_CTX)/docker-compose.yml down

logs: ## Логи сервиса
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f $(APP_NAME)

logs-mongo: ## Логи MongoDB
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f mongo

logs-kafka: ## Логи Kafka
	docker compose -f $(DOCKER_CTX)/docker-compose.yml logs -f kafka

ps: ## Статус контейнеров
	docker compose -f $(DOCKER_CTX)/docker-compose.yml ps