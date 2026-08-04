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
BACKUP_APP_GIT_COMMIT ?=

PROD_COMPOSE := docker compose -f docker-compose.yml --profile production
BACKUP_API_PROD_COMPOSE := docker compose -f docker-compose.yml -f docker-compose.backup-api.yml --profile production

.PHONY: dev db-gen migrate-up migrate-down migrate-up-docker migrate-down-docker migrate-force-docker swag lint lint-fix docker-up docker-dev-up docker-down docker-logs prod-validate-config prod-deploy prod-backup-api-validate-config prod-deploy-backup-api prod-status prod-logs backup-validate-paths backup-validate-config backup-run seed

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
# Order: validate -> build -> wait for db -> migrate up -> recreate app only -> wait for health.
prod-validate-config:
	@set -eu; \
		[ -n "$${PROD_DB_URL:-}" ] || { echo "PROD_DB_URL is required" >&2; exit 1; }; \
		[ -n "$${UPLOADS_HOST_PATH:-}" ] || { echo "UPLOADS_HOST_PATH is required" >&2; exit 1; }; \
		case "$${UPLOADS_HOST_PATH}" in /*) ;; *) echo "UPLOADS_HOST_PATH must be an absolute host path" >&2; exit 1;; esac; \
		[ -d "$${UPLOADS_HOST_PATH}" ] || { echo "UPLOADS_HOST_PATH must reference an existing directory" >&2; exit 1; }; \
		repo_path="$$(realpath -e -- "$(CURDIR)" 2>/dev/null)" || { echo "repository path could not be validated" >&2; exit 1; }; \
		uploads_path="$$(realpath -e -- "$${UPLOADS_HOST_PATH}" 2>/dev/null)" || { echo "UPLOADS_HOST_PATH could not be validated" >&2; exit 1; }; \
		if [ "$$uploads_path" = "$$repo_path" ] || [ "$${uploads_path#"$${repo_path}/"}" != "$$uploads_path" ]; then \
			echo "production uploads must not use a directory inside the repository" >&2; exit 1; \
		fi; \
		UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" $(PROD_COMPOSE) config --quiet

prod-deploy: prod-validate-config
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" $(PROD_COMPOSE) build app
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" $(PROD_COMPOSE) up -d --wait --no-recreate db
	@docker run --rm --network "$${DOCKER_NETWORK:-permatatex-shared-net}" -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" "$${MIGRATE_DOCKER_IMAGE:-migrate/migrate}" -path=/migrations -database "$${PROD_DB_URL}" up
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" $(PROD_COMPOSE) up -d --no-deps --wait --wait-timeout 60 app
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" $(PROD_COMPOSE) ps

# Opt-in production deployment with Backup API mounts and runtime configuration.
# This deploys the web application only; it never runs the manual backup service.
prod-backup-api-validate-config: backup-validate-paths
	@set -eu; \
		commit="$${BACKUP_APP_GIT_COMMIT:-}"; \
		[ "$${#commit}" -eq 40 ] && case "$$commit" in *[!0-9a-fA-F]*) false;; *) true;; esac || { echo "BACKUP_APP_GIT_COMMIT must be a 40-character hexadecimal commit hash" >&2; exit 1; }; \
		UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" BACKUP_HOST_PATH="$${BACKUP_HOST_PATH}" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$$commit" $(BACKUP_API_PROD_COMPOSE) config --quiet

prod-deploy-backup-api: prod-backup-api-validate-config
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" BACKUP_HOST_PATH="$${BACKUP_HOST_PATH}" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$${BACKUP_APP_GIT_COMMIT}" $(BACKUP_API_PROD_COMPOSE) build app
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" BACKUP_HOST_PATH="$${BACKUP_HOST_PATH}" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$${BACKUP_APP_GIT_COMMIT}" $(BACKUP_API_PROD_COMPOSE) up -d --wait --no-recreate db
	@docker run --rm --network "$${DOCKER_NETWORK:-permatatex-shared-net}" -v "$(CURDIR)/$(MIGRATIONS_PATH):/migrations" "$${MIGRATE_DOCKER_IMAGE:-migrate/migrate}" -path=/migrations -database "$${PROD_DB_URL}" up
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" BACKUP_HOST_PATH="$${BACKUP_HOST_PATH}" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$${BACKUP_APP_GIT_COMMIT}" $(BACKUP_API_PROD_COMPOSE) up -d --no-deps --wait --wait-timeout 60 app
	@UPLOADS_HOST_PATH="$${UPLOADS_HOST_PATH}" BACKUP_HOST_PATH="$${BACKUP_HOST_PATH}" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$${BACKUP_APP_GIT_COMMIT}" $(BACKUP_API_PROD_COMPOSE) ps

prod-status:
	docker compose ps app db

prod-logs:
	docker compose logs --tail 100 app

# The backup service is intentionally manual. Host mounts must already exist
# and stay outside the repository; do not place keys or packages in Git.
backup-validate-paths:
	@set -eu; \
		[ -n "$${PROD_DB_URL:-}" ] || { echo "PROD_DB_URL is required" >&2; exit 1; }; \
		[ -n "$${UPLOADS_HOST_PATH:-}" ] || { echo "UPLOADS_HOST_PATH is required" >&2; exit 1; }; \
		[ -n "$${BACKUP_HOST_PATH:-}" ] || { echo "BACKUP_HOST_PATH is required" >&2; exit 1; }; \
		[ -n "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH:-}" ] || { echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH is required" >&2; exit 1; }; \
		[ -n "$${BACKUP_GPG_RECIPIENT:-}" ] || { echo "BACKUP_GPG_RECIPIENT is required" >&2; exit 1; }; \
		case "$${UPLOADS_HOST_PATH}" in /*) ;; *) echo "UPLOADS_HOST_PATH must be an absolute host path" >&2; exit 1;; esac; \
		case "$${BACKUP_HOST_PATH}" in /*) ;; *) echo "BACKUP_HOST_PATH must be an absolute host path" >&2; exit 1;; esac; \
		case "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" in /*) ;; *) echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH must be an absolute host path" >&2; exit 1;; esac; \
		[ -d "$${UPLOADS_HOST_PATH}" ] || { echo "UPLOADS_HOST_PATH must reference an existing directory" >&2; exit 1; }; \
		[ -d "$${BACKUP_HOST_PATH}" ] || { echo "BACKUP_HOST_PATH must reference an existing directory" >&2; exit 1; }; \
		[ -f "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" ] || { echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH must reference an existing regular file" >&2; exit 1; }; \
		repo="$$(realpath -e -- "$(CURDIR)" 2>/dev/null)" || { echo "repository path could not be validated" >&2; exit 1; }; \
		uploads="$$(realpath -e -- "$${UPLOADS_HOST_PATH}" 2>/dev/null)" || { echo "UPLOADS_HOST_PATH could not be validated" >&2; exit 1; }; \
		backups="$$(realpath -e -- "$${BACKUP_HOST_PATH}" 2>/dev/null)" || { echo "BACKUP_HOST_PATH could not be validated" >&2; exit 1; }; \
		key="$$(realpath -e -- "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" 2>/dev/null)" || { echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH could not be validated" >&2; exit 1; }; \
		[ "$$uploads" != / ] || { echo "UPLOADS_HOST_PATH must not resolve to the filesystem root" >&2; exit 1; }; \
		[ "$$backups" != / ] || { echo "BACKUP_HOST_PATH must not resolve to the filesystem root" >&2; exit 1; }; \
		[ "$$uploads" != "$$backups" ] && [ "$${backups#"$$uploads"/}" = "$$backups" ] && [ "$${uploads#"$$backups"/}" = "$$uploads" ] || { echo "UPLOADS_HOST_PATH and BACKUP_HOST_PATH must not overlap" >&2; exit 1; }; \
		case "$$uploads" in "$$repo"|"$$repo"/*) echo "UPLOADS_HOST_PATH must be outside the repository" >&2; exit 1;; esac; \
		case "$$backups" in "$$repo"|"$$repo"/*) echo "BACKUP_HOST_PATH must be outside the repository" >&2; exit 1;; esac; \
		case "$$key" in "$$repo"|"$$repo"/*) echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH must be outside the repository" >&2; exit 1;; esac; \
		case "$$key" in "$$uploads"|"$$uploads"/*|"$$backups"|"$$backups"/*) echo "BACKUP_GPG_PUBLIC_KEY_HOST_PATH must be outside upload and backup directories" >&2; exit 1;; esac

backup-validate-config: backup-validate-paths
	@set -eu; uploads="$$(realpath -e -- "$${UPLOADS_HOST_PATH}" 2>/dev/null)"; backups="$$(realpath -e -- "$${BACKUP_HOST_PATH}" 2>/dev/null)"; key="$$(realpath -e -- "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}" 2>/dev/null)"; \
		UPLOADS_HOST_PATH="$$uploads" BACKUP_HOST_PATH="$$backups" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$$key" BACKUP_GPG_RECIPIENT="$${BACKUP_GPG_RECIPIENT}" PROD_DB_URL="$${PROD_DB_URL}" BACKUP_APP_GIT_COMMIT="$${BACKUP_APP_GIT_COMMIT:-unknown}" docker compose -f docker-compose.yml --profile backup config --quiet

backup-run: backup-validate-config
	@set -eu; uploads="$$(realpath -e -- "$${UPLOADS_HOST_PATH}")"; backups="$$(realpath -e -- "$${BACKUP_HOST_PATH}")"; key="$$(realpath -e -- "$${BACKUP_GPG_PUBLIC_KEY_HOST_PATH}")"; UPLOADS_HOST_PATH="$$uploads" BACKUP_HOST_PATH="$$backups" BACKUP_GPG_PUBLIC_KEY_HOST_PATH="$$key" docker compose --profile backup run --rm --no-deps backup run
