APP_NAME := permatatex-inventory
SQLC_CONFIG := sqlc.yaml
MIGRATIONS_PATH := db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:15432/permatatex_inventory?sslmode=disable
LINT_CONFIG := .golangci.yml

.PHONY: dev db-gen migrate-up migrate-down swag lint lint-fix docker-up docker-down docker-logs seed

dev:
	air -c .air.toml

seed:
	go run cmd/seeder/main.go

db-gen:
	sqlc generate -f $(SQLC_CONFIG)

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

swag:
	swag init -g main.go -d cmd/web,internal/delivery/http,internal/model,internal/entity,pkg/response -o docs --parseInternal --parseDependency

lint:
	golangci-lint run --config $(LINT_CONFIG) ./...

lint-fix:
	golangci-lint run --fix --config $(LINT_CONFIG) ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app
