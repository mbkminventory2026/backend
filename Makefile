APP_NAME := permatatex-inventory
SQLC_CONFIG := sqlc.yaml
MIGRATIONS_PATH := db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:5432/permatatex_inventory?sslmode=disable

.PHONY: dev db-gen migrate-up migrate-down swag

dev:
	air -c .air.toml

db-gen:
	sqlc generate -f $(SQLC_CONFIG)

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

swag:
	swag init -g cmd/web/main.go -o docs