# EffectiveMobile - Сервис агрегации подписок

REST-сервис для агрегации данных об онлайн-подписках пользователей, написанный на Go.

## Описание

Сервис предоставляет CRUD операции для подписок пользователей и рассчитывает суммарную стоимость всех подписок за выбранный период с фильтрацией по ID пользователя и названию подписки.

## Быстрый старт

### Требования
- Docker & Docker Compose
- Go 1.24+ (для разработки)

### Запуск сервиса

1. **Запуск сервиса через Docker Compose**:
   ```bash
   docker-compose up -d
   ```

2. **Сервис будет доступен по адресу**: `http://localhost:8080`

3. **API документация (Swagger UI)**: http://localhost:8080/swagger/

## API

Сервис предоставляет REST API для управления подписками:

### Основные эндпоинты

- **Подписки:** `/api/v1/subscriptions` - CRUD операции
- **Статистика:** `/api/v1/stats/total` - расчет суммарной стоимости

### Форматы данных

**Даты:** MM-YYYY (например: "01-2024", "12-2024")

Примечания по поведению дат:
- Если не заданы `start_date` и `end_date` при расчёте статистики, период считается от минимальной даты подписок до текущего месяца включительно (по первому числу месяца). Это сделано для удобства и может отличаться от ожиданий — при необходимости указывайте период явно.
- Очистка `end_date` при обновлении: чтобы сделать подписку бессрочной, передайте пустую строку `""` или строку `"null"` в поле `end_date`. JSON `null` трактуется как "поле отсутствует" и не изменяет текущее значение `end_date`.

**HTTP коды ответов:**
- `200` - Успешный запрос
- `201` - Ресурс создан
- `204` - Успешное удаление
- `400` - Неверные параметры
- `404` - Не найдено
- `409` - Конфликт (дубликат)
- `500` - Ошибка сервера

Подробная документация API с примерами доступна в **Swagger UI**: http://localhost:8080/swagger/

Генерация Swagger (обновление `.static/swagger/swagger.json`):
```bash
make gen_swagger
```

### Локальная разработка

1. **Запуск базы данных**:
   ```bash
   docker-compose up -d postgres
   ```

2. **Применение миграций**:
   ```bash
   go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations"
   ```

3. **Запуск сервиса**:
   ```bash
   go run cmd/subscription/main.go
   ```

### Использование Make команд

```bash
# Unit тесты (быстрые, с моками)
make test-unit

# Интеграционные тесты (нужна запущенная БД PostgreSQL на 5433)
make test-integration

# Все тесты
make test-all

# Применение миграций БД
make migrate-up
```

## Тестирование

Проект включает комплексное тестирование:

- **Unit тесты**: быстрые тесты с моками для API и сервисов
- **Интеграционные тесты**: реальные HTTP-запросы к запущенному сервису и реальной БД

Запуск на Windows (PowerShell):
```powershell
$env:INTEGRATION_TESTS="true"; go test -v ./tests/integration/...
```

Запуск всех тестов:
```powershell
$env:INTEGRATION_TESTS="true"; go test -v ./...
```

Перед интеграционными тестами поднимите инфраструктуру и примените миграции:
```powershell
docker-compose up -d
go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations"
```

## Архитектура проекта

Проект следует принципам Clean Architecture с разделением на слои:

### Структура каталогов
```
EffectiveMobile/
├── cmd/
│   ├── subscription/        # Точка входа приложения
│   └── migrator/           # Инструмент миграций
├── internal/
│   ├── api/
│   │   ├── handlers/       # HTTP обработчики (API слой)
│   │   └── middleware/     # Middleware (логирование и т.д.)
│   ├── service/            # Бизнес-логика
│   ├── repository/         # Работа с базой данных
│   └──  config/            # Конфигурация
├── pkg/
│   ├── api/response/      # HTTP ответы
│   └── postgres/          # PostgreSQL провайдер
├── migrations/            # SQL миграции
└── tests/
    └── integration/       # Интеграционные тесты
```

### Слои приложения

**API Layer (handlers)**
- Обработка HTTP запросов
- Валидация входных данных
- Формирование ответов

**Service Layer (service)**
- Бизнес-логика приложения
- Валидация бизнес-правил
- Координация между репозиториями

**Repository Layer (repository)**
- Взаимодействие с базой данных
- SQL запросы через squirrel
- Обработка ошибок БД

## Конфигурация

Сервис использует YAML файлы конфигурации. 

### Миграции базы данных

Миграции находятся в директории `migrations/`.

**При использовании Docker Compose:**
Миграции применяются автоматически при запуске контейнера через `docker-entrypoint.sh`.

**Для локальной разработки:**
```bash
# Вручную
go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "./migrations"

# Через Makefile
make migrate-up
```

### Shutdown

Приложение корректно обрабатывает сигналы завершения работы:

- **SIGINT** (Ctrl+C) - остановка в режиме разработки
- **SIGTERM** (docker stop) - остановка в Docker

При получении сигнала:
1. ✅ Останавливается прием новых запросов
2. ✅ Дожидается завершения активных запросов
3. ✅ Закрывает соединения с базой данных
4. ✅ Корректно завершает работу

## Генерация моков

Используем `go.uber.org/mock`:
```bash
go run go.uber.org/mock/mockgen@latest -source=internal/service/subscription.go -destination=internal/service/subscription_mock.go -package=service
go run go.uber.org/mock/mockgen@latest -source=internal/service/stats.go -destination=internal/service/stats_mock.go -package=service
```