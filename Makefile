APP_NAME := permatatex-inventory
SQLC_CONFIG := sqlc.yaml
MIGRATIONS_PATH := db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:15432/permatatex_inventory?sslmode=disable
LINT_CONFIG := .golangci.yml
MIGRATE_DOCKER_IMAGE ?= migrate/migrate
DOCKER_DB_URL ?= postgres://postgres:postgres@permatatex-postgres:5432/permatatex_inventory?sslmode=disable
DOCKER_NETWORK ?= permatatex-shared-net
PROD_DB_URL ?=
UPLOADS_HOST_PATH ?=
BACKUP_HOST_PATH ?=
BACKUP_GPG_PUBLIC_KEY_HOST_PATH ?=
BACKUP_GPG_RECIPIENT ?=

.PHONY: dev db-gen migrate-up migrate-down migrate-up-docker migrate-down-docker migrate-force-docker swag lint lint-fix docker-up docker-dev-up docker-down docker-logs prod-validate-config prod-deploy prod-status prod-logs backup-validate-config backup-run seed

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
	docker compose up -d --wait db
	@docker run --rm --network $(DOCKER_NETWORK) -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" $(MIGRATE_DOCKER_IMAGE) -path=/migrations -database "$(DOCKER_DB_URL)" up

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
	$(MAKE) docker-dev-up

docker-dev-up:
	docker compose up --build -d dev

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f dev

# Production-only safe redeploy. The database URL and persistent uploads host
# path must be supplied explicitly through the production environment.
# Order: validate -> build -> migrate up -> recreate app only -> wait for health.
prod-validate-config:
	$(if $(strip $(PROD_DB_URL)),,$(error PROD_DB_URL is required))
	$(if $(strip $(UPLOADS_HOST_PATH)),,$(error UPLOADS_HOST_PATH is required))
	$(if $(filter /%,$(strip $(UPLOADS_HOST_PATH))),,$(error UPLOADS_HOST_PATH must be an absolute host path))
	$(if $(wildcard $(strip $(UPLOADS_HOST_PATH))/.),,$(error UPLOADS_HOST_PATH must already exist))
	@repo_path="$$(realpath "$(CURDIR)")"; uploads_path="$$(realpath "$(UPLOADS_HOST_PATH)")"; \
		if [ "$$uploads_path" = "$$repo_path" ] || [ "$${uploads_path#"$${repo_path}/"}" != "$$uploads_path" ]; then \
			echo "production uploads must not use a directory inside the repository" >&2; exit 1; \
		fi
	@UPLOADS_HOST_PATH="$(UPLOADS_HOST_PATH)" docker compose --profile production config --quiet

prod-deploy: prod-validate-config
	UPLOADS_HOST_PATH="$(UPLOADS_HOST_PATH)" docker compose build app
	UPLOADS_HOST_PATH="$(UPLOADS_HOST_PATH)" docker compose up -d --wait --no-recreate db
	@docker run --rm --network $(DOCKER_NETWORK) -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" $(MIGRATE_DOCKER_IMAGE) -path=/migrations -database "$(PROD_DB_URL)" up
	UPLOADS_HOST_PATH="$(UPLOADS_HOST_PATH)" docker compose up -d --no-deps --wait --wait-timeout 60 app
	UPLOADS_HOST_PATH="$(UPLOADS_HOST_PATH)" docker compose ps

prod-status:
	docker compose ps app db

prod-logs:
	docker compose logs --tail 100 app

# The backup service is intentionally manual. Host mounts must already exist
# and stay outside the repository; do not place keys or packages in Git.
backup-validate-config:
	$(if $(strip $(PROD_DB_URL)),,$(error PROD_DB_URL is required))
	$(if $(strip $(UPLOADS_HOST_PATH)),,$(error UPLOADS_HOST_PATH is required))
	$(if $(strip $(BACKUP_HOST_PATH)),,$(error BACKUP_HOST_PATH is required))
	$(if $(strip $(BACKUP_GPG_PUBLIC_KEY_HOST_PATH)),,$(error BACKUP_GPG_PUBLIC_KEY_HOST_PATH is required))
	$(if $(strip $(BACKUP_GPG_RECIPIENT)),,$(error BACKUP_GPG_RECIPIENT is required))
	$(if $(filter /%,$(strip $(UPLOADS_HOST_PATH))),,$(error UPLOADS_HOST_PATH must be an absolute host path))
	$(if $(filter /%,$(strip $(BACKUP_HOST_PATH))),,$(error BACKUP_HOST_PATH must be an absolute host path))
	$(if $(filter /%,$(strip $(BACKUP_GPG_PUBLIC_KEY_HOST_PATH))),,$(error BACKUP_GPG_PUBLIC_KEY_HOST_PATH must be an absolute host path))
	@set -eu; repo="$$(realpath -e "$(CURDIR)")"; uploads="$$(realpath -e "$(UPLOADS_HOST_PATH)")"; backups="$$(realpath -e "$(BACKUP_HOST_PATH)")"; key="$$(realpath -e "$(BACKUP_GPG_PUBLIC_KEY_HOST_PATH)")"; \
		[ "$$uploads" != / ] && [ "$$backups" != / ] && [ -d "$$uploads" ] && [ -d "$$backups" ] && [ -f "$$key" ] || { echo "unsafe backup host path" >&2; exit 1; }; \
		[ "$$uploads" != "$$backups" ] && [ "$${backups#"$$uploads"/}" = "$$backups" ] && [ "$${uploads#"$$backups"/}" = "$$uploads" ] || { echo "backup and uploads paths overlap" >&2; exit 1; }; \
		case "$$uploads" in "$$repo"|"$$repo"/*) echo "uploads path must be outside repository" >&2; exit 1;; esac; case "$$backups" in "$$repo"|"$$repo"/*) echo "backup path must be outside repository" >&2; exit 1;; esac; case "$$key" in "$$uploads"|"$$uploads"/*|"$$backups"|"$$backups"/*) echo "public key must be outside backup storage" >&2; exit 1;; esac; \
		UPLOADS_HOST_PATH="$$uploads" BACKUP_HOST_PATH="$$backups" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$$key" BACKUP_GPG_RECIPIENT="$(BACKUP_GPG_RECIPIENT)" PROD_DB_URL="$(PROD_DB_URL)" docker compose --profile backup config --quiet

backup-run: backup-validate-config
	@set -eu; uploads="$$(realpath -e "$(UPLOADS_HOST_PATH)")"; backups="$$(realpath -e "$(BACKUP_HOST_PATH)")"; key="$$(realpath -e "$(BACKUP_GPG_PUBLIC_KEY_HOST_PATH)")"; UPLOADS_HOST_PATH="$$uploads" BACKUP_HOST_PATH="$$backups" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$$key" docker compose --profile backup run --rm --no-deps backup run
