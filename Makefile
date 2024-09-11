build:
	@echo "Building..."
	@GOFLAGS="-buildvcs=true" go build -o bin/dbank
	@echo "Build complete"

