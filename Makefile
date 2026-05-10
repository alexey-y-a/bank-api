# .PHONY говорит Make, что эти цели не являются файлами — выполнять всегда
.PHONY: help build tidy test test-integration run docker-up docker-down migrate-up migrate-down lint

# Выводит список доступных команд с описанием
help:
	@echo "Available targets:"
	@echo "  make build          - Собрать бинарный файл"
	@echo "  make tidy           - Обновить go.mod и go.sum"
	@echo "  make test           - Запустить unit-тесты (-short)"
	@echo "  make test-integration - Запустить интеграционные тесты"
	@echo "  make run            - Запустить приложение локально"
	@echo "  make docker-up      - Запустить PostgreSQL в Docker"
	@echo "  make docker-down    - Остановить контейнеры"
	@echo "  make migrate-up     - Накатить миграции (требует goose)"
	@echo "  make migrate-down   - Откатить последнюю миграцию"
	@echo "  make lint           - Проверить код через golangci-lint"

# Компилирует приложение в бинарный файл 'bank-api' в корне проекта
# -o указывает имя выходного файла
build:
	go build -o bank-api ./cmd/api

# go mod tidy удаляет неиспользуемые зависимости и добавляет недостающие
# Важно запускать после добавления новых импортов
tidy:
	go mod tidy

# -short пропускает медленные интеграционные тесты
# -v включает подробный вывод
# ./... означает "все пакеты рекурсивно"
test:
	go test ./... -short -v

# Интеграционные тесты требуют Docker (testcontainers)
# -run Integration запускает только тесты с "Integration" в имени
# -count=1 отключает кэш тестов — важно для стабильности
test-integration:
	go test ./... --run Integration -v -count=1

# go run компилирует и запускает код без создания бинарника (для разработки)
run:
	go run ./cmd/api

# -f deploy/docker-compose.yml указывает Compose искать файл в папке deploy/
# -d запускает сервисы в фоновом режиме (detached)docker-up:
docker-up:
	docker compose -f deploy/docker-compose.yml up -d

# Останавливает контейнеры, удаляет сеть, но сохраняет volume (данные БД)
docker-down:
	docker compose -f deploy/docker-compose.yml down

# Накатывает все "up" миграции из папки migrations/
# Требует установленного goose: go install github.com/pressly/goose/v3/cmd/goose@latest
migrate-up:
	goose -dir migrations postgres "${DB_DSN}" up

# Откатывает последнюю миграцию ("down")
migrate-down:
	goose -dir migrations postgres "${DB_DSN}" down

# Проверяет код на соответствие стандартам и лучшим практикам
# Требует установленного golangci-lint: https://golangci-lint.run/
lint:
	golangci-lint run ./...