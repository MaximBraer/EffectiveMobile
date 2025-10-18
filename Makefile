.PHONY: test test-unit test-e2e test-all docker-up docker-down migrate-up migrate-down test-e2e-full reset-db clean

test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/http-server/handlers/subscription/save

test-e2e:
	@echo "Running E2E tests..."
	go test -v ./tests/e2e

migrate-up:
	@echo "Applying database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations"

migrate-down:
	@echo "Rolling back database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations" -down

test-all: test-unit test-e2e

test-short:
	@echo "Running short tests..."
	go test -short -v ./...

docker-up:
	@echo "Starting Docker environment..."
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 10

docker-down:
	@echo "Stopping Docker environment..."
	docker-compose down

test-e2e-full: docker-up migrate-up test-e2e docker-down

reset-db: docker-down clean docker-up migrate-up

clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -rf pgdata/
