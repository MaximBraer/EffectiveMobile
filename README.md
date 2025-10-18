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
  "service_name": "{service_name}",
  "price": {price},
  "user_id": "{user_id}",
  "start_date": "{start_date}",
  "end_date": "{end_date}" // опционально
}
```

**Ответ:**
```json
{
  "status": "ok",
  "id": {id}
}
```

#### Получение подписки
```http
GET /api/v1/subscriptions/{id}
```

**Ответ:**
```json
{
  "id": {id},
  "service_name": "{service_name}",
  "price": {price},
  "user_id": "{user_id}",
  "start_date": "{start_date}",
  "end_date": "{end_date}"
}
```

#### Обновление подписки
```http
PUT /api/v1/subscriptions/{id}
Content-Type: application/json

{
  "price": {price},
  "end_date": "{end_date}"
}
```

**Ответ:**
```json
{
  "status": "ok"
}
```

#### Удаление подписки
```http
DELETE /api/v1/subscriptions/{id}
```

**Ответ:**
```json
{
  "status": "ok"
}
```

#### Список подписок
```http
GET /api/v1/subscriptions?limit={limit}&offset={offset}&user_id={user_id}&service_name={service_name}
```

**Ответ:**
```json
{
  "subscriptions": [
    {
      "id": {id},
      "service_name": "{service_name}",
      "price": {price},
      "user_id": "{user_id}",
      "start_date": "{start_date}",
      "end_date": "{end_date}"
    }
  ],
  "total": {total}
}
```

### Статистика

#### Получение суммарной стоимости подписок
```http
GET /api/v1/stats/total?user_id={user_id}&start_date={start_date}&end_date={end_date}&service_name={service_name}
```

**Ответ:**
```json
{
  "total_cost": {total_cost},
  "subscriptions_count": {subscriptions_count},
  "period": {
    "start_date": "{start_date}",
    "end_date": "{end_date}"
  }
}
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