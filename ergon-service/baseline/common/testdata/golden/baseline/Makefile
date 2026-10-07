# Managed by ergon init. Add repository settings to .ergon/local/Makefile and run ergon init sync.
#
# make check is the gate of the repository. Each language adds its own fmt, lint, test, audit and
# check targets as prerequisites of the targets below, and CI runs make check-<language> per
# language. Every target runs its tools through ergon tool run, which installs the version that
# .ergon.yaml names.

.DEFAULT_GOAL := check

# The ergon that runs the tools of the targets.
ERGON ?= ergon

.PHONY: help fmt lint test audit check

help: ## List the targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z][a-z-]*:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format the sources of every language
lint: ## Lint the sources of every language
test: ## Run the tests of every language
audit: ## Run the vulnerability scan of every language
check: ## Run the gate of every language
