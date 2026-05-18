BINARY_NAME := server
BUILD_DIR   := build
CMD         := .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: build run check test test-integration lint fmt vuln install-tools

## build: compile the source code
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD)

## run: run the server without building 
run:
	go run $(CMD)

## check: run fmt, lint, vuln, and test-integration targets
check: test-integration lint fmt vuln

## test: run unit tests
test:
	go test -race -shuffle=on -timeout=5m ./...

## test-integration: run unit and integration tests (requires Docker)
test-integration:
	go test -tags=integration -race -shuffle=on -timeout=5m ./...

## lint: run linters
lint:
	golangci-lint run ./...

## fmt: run formatters
fmt:
	golangci-lint fmt --diff-colored ./...

## vuln: scan for vulnerabilities
vuln:
	govulncheck ./...

## install-tools: install required dev tools 
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"