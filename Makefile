# Managed by ergon init. Add repository settings to .ergon/local/Makefile and run ergon init sync.
#
# make check is the gate of the repository. Each language adds its own fmt, lint, test, audit and
# check targets as prerequisites of the targets below, and CI runs make check-<language> per
# language.

.DEFAULT_GOAL := check

.PHONY: help fmt lint test audit check

help: ## List the targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z][a-z-]*:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format the sources of every language
lint: ## Lint the sources of every language
test: ## Run the tests of every language
audit: ## Run the vulnerability scan of every language
check: ## Run the gate of every language

# Go

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0

.PHONY: fmt-go lint-go test-go audit-go check-go
fmt: fmt-go
lint: lint-go
test: test-go
audit: audit-go
check: check-go

fmt-go: ## Format the Go sources of every module with the formatters of .golangci.yml
	@go list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "golangci-lint fmt $$dir"; (cd "$$dir" && $(GOLANGCI_LINT) fmt ./...) || exit 1; done
lint-go: ## Lint and format-check the Go sources of every module with .golangci.yml
	@go list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "golangci-lint $$dir"; (cd "$$dir" && $(GOLANGCI_LINT) run ./... && $(GOLANGCI_LINT) fmt --diff ./...) || exit 1; done
test-go: ## Run the Go tests of every module
	@go list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "go test $$dir"; go -C "$$dir" test ./... || exit 1; done
audit-go: ## Scan every Go module for known vulnerabilities that its code reaches
	@go list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "govulncheck $$dir"; go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -C "$$dir" ./... || exit 1; done
check-go: lint-go test-go audit-go ## Run the gate of Go
