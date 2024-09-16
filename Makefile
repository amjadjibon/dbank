.PHONY: build lint serve migrate-up migrate-down start-db-local stop-db-local gen
build: lint
	@echo "Building..."
	@CGO_ENABLED=0 GOFLAGS="-buildvcs=true" go build -o bin/dbank
	@echo "Build complete"

gen:
	@echo "Generating..."
	@cd proto && buf generate && cd ..
	@echo "Generation complete"

lint:
	@echo "Linting..."
	@golangci-lint run --fix
	@echo "Lint complete"

serve: build
	@echo "Starting server..."
	@./bin/dbank serve

migrate-up: build
	@echo "Migrating up..."
	@./bin/dbank migrate up
	@echo "Migrations complete"

migrate-down: build
	@echo "Migrating down..."
	@./bin/dbank migrate down
	@echo "Migrations complete"

start-db-local:
	@docker-compose -f compose.yml up -d
	@echo "Database started"

stop-db-local:
	@docker-compose -f compose.yml down --volumes --remove-orphans
	@echo "Database stopped"
