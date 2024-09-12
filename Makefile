build: lint
	@echo "Building..."
	@CGO_ENABLED=0 GOFLAGS="-buildvcs=true" go build -o bin/dbank
	@echo "Build complete"

lint:
	@echo "Linting..."
	@golangci-lint run --fix
	@echo "Lint complete"
