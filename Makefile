# Makefile

# Update to include stack mk files
include stacks/basic/stack.mk
include stacks/template/stack.mk

# Ensure help shows stack targets
help: 
	@echo "Stack Targets:"
	@echo "	f_stack:  Run stack tests"

# Example target descriptions
f_stack:

	# Description: Run stack tests
	@echo "Running stack tests..."