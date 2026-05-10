BINARY_NAME := server
BUILD_DIR   := build
CMD         := .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: build run check test test-integration lint fmt vet security install-tools

## build: compile the source code
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD)

## run: run the server without compiling
run:
	go run $(CMD)

## check: run fmt, vet, lint, security, and test targets
check: fmt vet lint security test

## test: run all tests with race detector
test:
	go test ./... -v -race

## test-integration: run unit and integration tests (requires Docker)
test-integration:
	go test -tags=integration ./... -v -race

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: check if codebase is compliant with goimports' formatting
fmt:
	goimports -l .

## vet: run go vet
vet:
	go vet ./...

## security: scan for vulnerabilities
security:
	govulncheck ./...
	gosec -exclude-generated ./...

## install-tools: install required tools for check target
install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"