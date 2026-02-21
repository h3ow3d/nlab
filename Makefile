.DEFAULT_GOAL := help

.PHONY: help build install test lint fmt

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

install: ## Install nlab to ~/.local/bin (adds to PATH if needed)
	@mkdir -p "$(HOME)/.local/bin"
	go build -o "$(HOME)/.local/bin/nlab" ./cmd/nlab
	@echo ""
	@echo "nlab installed to $(HOME)/.local/bin/nlab"
	@echo ""
	@if ! echo "$$PATH" | grep -q "$(HOME)/.local/bin"; then \
		echo "  NOTE: $(HOME)/.local/bin is not in your PATH."; \
		echo "  Add the following line to your shell profile (~/.bashrc or ~/.zshrc):"; \
		echo "    export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
		echo "  Then restart your shell or run:"; \
		echo "    source ~/.bashrc"; \
		echo ""; \
	fi
	@echo "Run 'nlab doctor' to verify prerequisites."

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go source files
	gofmt -w ./...

test: ## Run Go tests
	go test ./...
