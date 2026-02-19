.DEFAULT_GOAL := help

STACK_FILES := $(wildcard stacks/*/stack.mk)
include $(STACK_FILES)

.PHONY: help list

help: ## Show this help message
	@echo ""
	@echo "red-team lab framework"
	@echo "-----------------------"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""

list: ## List all libvirt domains
	virsh list --all
