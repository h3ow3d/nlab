.DEFAULT_GOAL := help

STACK ?= basic

.PHONY: help build test up down list

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

test: ## Run Go tests
	go test ./...

up: ## Bring up a stack (STACK=<name>, default: basic)
	./nlab up $(STACK)

down: ## Tear down a stack (STACK=<name>, default: basic)
	./nlab down $(STACK)

list: ## List all libvirt domains
	./nlab list
