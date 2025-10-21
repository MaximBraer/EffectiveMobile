.PHONY: test-unit test-integration test-all docker-up docker-down migrate-up migrate-down reset-db gen_swagger

test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/...

test-integration:
	@echo "Running integration tests..."
	INTEGRATION_TESTS=true go test -v ./tests/integration/...

test-all: test-unit test-integration

migrate-up:
	@echo "Applying database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations"

migrate-down:
	@echo "Rolling back database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/subscriptions?sslmode=disable" -migrations-path "migrations" -down

docker-up:
	@echo "Starting Docker environment..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker environment..."
	docker-compose down

gen_swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init --requiredByDefault --parseDependency --parseInternal --parseDepth 2 --parseGoList --output=./.static/swagger --outputTypes=json -g ./cmd/subscription/main.go

