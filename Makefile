# Loads .env so DATABASE_URL etc. are available to make targets.
-include .env
export

GOOSE_VERSION  := v3.28.0
MIGRATIONS_DIR := internal/database/migrations
GOOSE          := go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)"

.PHONY: run build test test-db vet fmt tidy db-up db-down db-logs db-shell migrate-up migrate-down migrate-status migrate-create

## Application
run:            ## Run the API locally
	go run ./cmd/api

build:          ## Build binary to bin/api
	go build -o bin/api ./cmd/api

test:           ## Run unit tests with the race detector (DB tests skipped)
	go test -race ./...

test-db:        ## Run all tests including Postgres integration tests (needs db-up)
	TEST_DATABASE_URL="$(DATABASE_URL)" go test -race ./...

vet:            ## Static checks
	go vet ./...

fmt:            ## Format code
	gofmt -l -w .

tidy:           ## Tidy go.mod/go.sum
	go mod tidy

## Database (Docker)
db-up:          ## Start Postgres and wait until healthy
	docker compose up -d --wait db

db-down:        ## Stop Postgres (data kept in volume)
	docker compose down

db-logs:        ## Tail Postgres logs
	docker compose logs -f db

db-shell:       ## psql into the database
	docker compose exec db psql -U dashboard -d dashboard

## Migrations (the app also applies these on startup)
migrate-up:     ## Apply pending migrations
	$(GOOSE) up

migrate-down:   ## Roll back the last migration
	$(GOOSE) down

migrate-status: ## Show migration status
	$(GOOSE) status

migrate-create: ## Create a new migration: make migrate-create name=add_thing
	$(GOOSE) create $(name) sql
