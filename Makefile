APP_NAME := permatatex-inventory
SQLC_CONFIG := sqlc.yaml
MIGRATIONS_PATH := db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:15432/permatatex_inventory?sslmode=disable
LINT_CONFIG := .golangci.yml
MIGRATE_DOCKER_IMAGE ?= migrate/migrate
DOCKER_DB_URL ?= postgres://postgres:postgres@permatatex-postgres:5432/permatatex_inventory?sslmode=disable
DOCKER_NETWORK ?= backend_permatatex-net

.PHONY: dev db-gen migrate-up migrate-down migrate-up-docker migrate-down-docker migrate-force-docker swag lint lint-fix docker-up docker-down docker-logs seed

dev:
	air -c .air.toml

seed:
	docker compose exec dev go run cmd/seeder/main.go

db-gen:
	sqlc generate -f $(SQLC_CONFIG)

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

migrate-up-docker:
	docker compose up -d db
	docker run --rm --network $(DOCKER_NETWORK) -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" $(MIGRATE_DOCKER_IMAGE) -path=/migrations -database "$(DOCKER_DB_URL)" up

migrate-down-docker:
	docker compose up -d db
	docker run --rm --network $(DOCKER_NETWORK) -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" $(MIGRATE_DOCKER_IMAGE) -path=/migrations -database "$(DOCKER_DB_URL)" down 1

migrate-force-docker:
	@if not defined VERSION (echo Usage: make migrate-force-docker VERSION^=^<n^> & exit /b 1)
	docker compose up -d db
	docker run --rm --network $(DOCKER_NETWORK) -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" $(MIGRATE_DOCKER_IMAGE) -path=/migrations -database "$(DOCKER_DB_URL)" force $(VERSION)

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
