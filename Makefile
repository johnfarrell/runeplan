.PHONY: all build run test lint fmt generate clean tools help

# Default target
all: generate build

## Build

generate: ## Run templ code generation (required before build)
	templ generate

build: ## Build the server binary
	go build ./...

run: ## Run the server (requires DATABASE_URL env var)
	go run ./cmd/server/main.go

## Testing

test: ## Run all tests
	go test ./...

test-pkg: ## Run tests for a specific package (usage: make test-pkg PKG=./domain/skill/...)
	go test $(PKG)

test-verbose: ## Run all tests with verbose output
	go test -v ./...

## Code Quality

fmt: ## Format all Go source files
	gofmt -w .

lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

## Dependencies

deps: ## Download and verify Go module dependencies
	go mod download
	go mod verify

tidy: ## Tidy go.mod and go.sum (only after all imports are in code)
	go mod tidy

## Tools

tools: ## Install required development tools (templ, golangci-lint)
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## Cleanup

clean: ## Remove generated _templ.go files and build artifacts
	find . -name '*_templ.go' -not -path './.git/*' -delete
	go clean ./...

## Help

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
