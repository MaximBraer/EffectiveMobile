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

2. **Сервис будет доступен по адресу**: `http://localhost:8082`

3. **Документация API**: `http://localhost:8082/swagger/` (Swagger UI)

## API Эндпоинты

### Управление подписками (CRUDL операции)

#### Создание подписки
```http
POST /api/v1/subscriptions
Content-Type: application/json

{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025",
  "end_date": "12-2025"  // опционально
}
```

#### Получение подписки
```http
GET /api/v1/subscriptions/{id}
```

#### Обновление подписки
```http
PUT /api/v1/subscriptions/{id}
Content-Type: application/json

{
  "price": 500,
  "end_date": "12-2025"
}
```

#### Удаление подписки
```http
DELETE /api/v1/subscriptions/{id}
```

#### Список подписок
```http
GET /api/v1/subscriptions?limit=10&offset=0&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Yandex
```

### Статистика

#### Получение суммарной стоимости подписок
```http
GET /api/v1/stats/total?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&start_date=01-2025&end_date=12-2025&service_name=Yandex
```

**Параметры запроса:**
- `user_id` (опционально) - Фильтр по ID пользователя
- `service_name` (опционально) - Фильтр по названию сервиса
- `start_date` (опционально) - Начало периода (формат ММ-ГГГГ)
- `end_date` (опционально) - Конец периода (формат ММ-ГГГГ)

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

# E2E тесты (медленные, с реальной БД)
make test-e2e

# Полный цикл E2E тестирования
make test-e2e-full

# Применение миграций БД
make migrate-up

# Сброс базы данных
make reset-db
```

## Тестирование

Проект включает комплексное тестирование:

- **Unit тесты**: Быстрые тесты с моками для отдельных компонентов
- **E2E тесты**: Полные интеграционные тесты с реальной базой данных
- **Покрытие тестами**: Все обработчики и бизнес-логика покрыты тестами

## Конфигурация

Сервис использует YAML файлы конфигурации. 

### Миграции базы данных

Миграции находятся в директории `migrations/`. Используйте инструмент миграций для их применения:

```bash
go run cmd/migrator/main.go -dsn "your-dsn" -migrations-path "migrations"
```