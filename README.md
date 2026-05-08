# Bank API

Полнофункциональный банковский backend на Go 1.26 для управления пользователями, счетами, картами, переводами и кредитами.

## Цель проекта
- Реализовать чистую архитектуру (Clean Architecture) с разделением слоёв

### Требования
- Go 1.26+
- Docker + Docker Compose (для PostgreSQL и других сервисов)
- Make (для удобства)

### Установка
```bash
# Клонируем репозиторий
git clone https://github.com/alexey-y-a/bank-api.git
cd bank-api

# Создаём локальный конфиг
.env
# Отредактируйте .env, подставив реальные значения

# Устанавливаем зависимости
go mod tidy

# Запускаем базу данных в Docker
make docker-up

# Накатываем миграции
make migrate-up

# Запускаем приложение
make run
```

### Проверка
```bash
# Health check
curl http://localhost:8080/healthz
# Ответ: {"status":"ok"}
```

### Сборка и запуск
```bash
# Собрать бинарный файл
make build

# Запустить локально
make run

# Запустить весь стек в Docker
make docker-up
```