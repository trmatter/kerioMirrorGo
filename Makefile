.PHONY: build test test-verbose test-coverage clean run

# Build the application
build:
	go build -o kerio-mirror-go.exe ./cmd/server

# Run all tests
test:
	go test ./...

# Run all tests with verbose output
test-verbose:
	go test ./... -v

# Run tests with coverage report
test-coverage:
	go test ./... -cover

# Run tests with detailed coverage report
test-coverage-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	go test ./... -bench=. -benchmem

# Clean build artifacts
clean:
	rm -f kerio-mirror-go.exe kerio-mirror-go
	rm -f coverage.out coverage.html

# Run the application
run: build
	./kerio-mirror-go.exe

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint installed)
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=windows GOARCH=amd64 go build -o kerio-mirror-go-windows-amd64.exe ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o kerio-mirror-go-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o kerio-mirror-go-darwin-amd64 ./cmd/server

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  test               - Run all tests"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  bench              - Run benchmarks"
	@echo "  clean              - Remove build artifacts"
	@echo "  run                - Build and run the application"
	@echo "  deps               - Download and tidy dependencies"
	@echo "  fmt                - Format source code"
	@echo "  lint               - Run linter"
	@echo "  build-all          - Build for multiple platforms"