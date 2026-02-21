.DEFAULT_GOAL := help

.PHONY: help build test lint fmt

help: ## Show this help message
	@echo ""
	@echo "nlab — Red-Team Lab Framework"
	@echo "------------------------------"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""

build: ## Build the nlab binary
	go build -o nlab ./cmd/nlab

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go source files
	gofmt -w ./...

test: ## Run Go tests
	go test ./...
