
# Go
#
# The Go targets run in every module that go list -m lists, which is every module of go.work. The
# section go of .ergon.yaml sets their tools and options, and the steps of check-go. One run
# overrides an option on the command line, such as make fuzz-go GO_FUZZ_MATCH=FuzzDecode.

# The go command.
GO ?= go

# The tools of the Go targets, which ergon installs at the versions of the section go.
GOLANGCI_LINT := $(ERGON) tool run go.golangci-lint --
GOVULNCHECK := $(ERGON) tool run go.govulncheck --
BENCHSTAT := $(ERGON) tool run go.benchstat --
DOKIMI_MUTATE_GO := $(ERGON) tool run go.dokimi-mutate-go --
ERGON_GO_VET := $(ERGON) tool run go.ergon-go-vet --

# The options of the section go.
GO_PATHS ?= ./...
GO_LINT_EXCLUDE ?=
GO_TEST_ARGS ?= -count=1
GO_RACE_ARGS ?= -count=1 -p=1
GO_FUZZ_MATCH ?= .
GO_FUZZ_TIME ?= 30s
GO_FUZZ_ARGS ?= -fuzzminimizetime=5s
GO_BENCH_MATCH ?= .
GO_BENCH_TIME ?= 1s
GO_BENCH_COUNT ?= 6
GO_BENCH_ARGS ?= -benchmem
GO_MUTATE_WORKERS ?= 1
GO_MUTATE_TIMEOUT ?= 0s
GO_MUTATE_ARGS ?=
GO_GENERATE_ARGS ?=
GO_AUDIT_ARGS ?=

# The files of two runs of bench-go that benchstat-go compares.
GO_BENCH_OLD ?=
GO_BENCH_NEW ?=

.PHONY: fmt-go lint-go test-go race-go fuzz-go bench-go benchstat-go mutate-go generate-go audit-go check-go
fmt: fmt-go
lint: lint-go
test: test-go
audit: audit-go
check: check-go

fmt-go: ## Format the Go sources of every module with the formatters of .golangci.yml
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "golangci-lint fmt $$dir"; \
		(cd "$$dir" && $(GOLANGCI_LINT) fmt $(GO_PATHS)) || exit 1; done
lint-go: ## Lint every module with .golangci.yml and ergon-go-vet, and check its format
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "golangci-lint $$dir"; \
		(cd "$$dir" && $(GOLANGCI_LINT) run $(GO_PATHS) && $(GOLANGCI_LINT) fmt --diff $(GO_PATHS) \
			&& $(ERGON_GO_VET) $(addprefix -exclude=,$(GO_LINT_EXCLUDE)) $(GO_PATHS)) || exit 1; done
test-go: ## Run the Go tests of every module
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "go test $$dir"; \
		$(GO) -C "$$dir" test $(GO_TEST_ARGS) $(GO_PATHS) || exit 1; done
race-go: ## Run the Go tests of every module under the race detector
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "go test -race $$dir"; \
		$(GO) -C "$$dir" test -race $(GO_RACE_ARGS) $(GO_PATHS) || exit 1; done
fuzz-go: ## Fuzz each fuzz target of every module whose name matches GO_FUZZ_MATCH, for GO_FUZZ_TIME each
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do \
		list="$$($(GO) -C "$$dir" test -list '$(GO_FUZZ_MATCH)' $(GO_PATHS))" || { printf '%s\n' "$$list"; exit 1; }; \
		targets="$$(printf '%s\n' "$$list" | awk '/^Fuzz/ { name[n++] = $$1; next } /^ok/ { for (i = 0; i < n; i++) print $$2, name[i]; n = 0 }')"; \
		if [ -z "$$targets" ]; then echo "no fuzz target matches $(GO_FUZZ_MATCH) in $$dir"; continue; fi; \
		printf '%s\n' "$$targets" | while read -r pkg name; do echo "go test -fuzz $$name $$pkg"; \
			$(GO) -C "$$dir" test -run '^$$' -fuzz "^$$name\$$" -fuzztime $(GO_FUZZ_TIME) $(GO_FUZZ_ARGS) "$$pkg" || exit 1; \
		done || exit 1; \
	done
bench-go: ## Run the benchmarks of every module whose names match GO_BENCH_MATCH, in the format of benchstat
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "go test -bench $$dir"; \
		$(GO) -C "$$dir" test -run '^$$' -bench '$(GO_BENCH_MATCH)' -benchtime $(GO_BENCH_TIME) \
			-count $(GO_BENCH_COUNT) $(GO_BENCH_ARGS) $(GO_PATHS) || exit 1; done
benchstat-go: ## Compare the results of bench-go in GO_BENCH_OLD with those in GO_BENCH_NEW
	@if [ -z "$(GO_BENCH_OLD)" ] || [ -z "$(GO_BENCH_NEW)" ]; then \
		echo "benchstat-go: set GO_BENCH_OLD and GO_BENCH_NEW to the files of two runs of bench-go" >&2; exit 2; fi
	$(BENCHSTAT) $(GO_BENCH_OLD) $(GO_BENCH_NEW)
mutate-go: ## Run dokimi-mutate-go on every module, and fail on a mutant that the tests miss
	@$(GO) list -m -f '{{.Dir}}' | { status=0; while IFS= read -r dir; do echo "dokimi-mutate-go $$dir"; \
		$(DOKIMI_MUTATE_GO) -C "$$dir" -workers $(GO_MUTATE_WORKERS) -timeout $(GO_MUTATE_TIMEOUT) $(GO_MUTATE_ARGS) \
			$(GO_PATHS) || status=1; done; exit $$status; }
generate-go: ## Run go generate in every module, and fail when it changes a file of the repository
	@before="$$(git diff HEAD --binary 2>/dev/null; git ls-files --others --exclude-standard)"; \
	$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "go generate $$dir"; \
		$(GO) -C "$$dir" generate $(GO_GENERATE_ARGS) $(GO_PATHS) || exit 1; done || exit 1; \
	after="$$(git diff HEAD --binary 2>/dev/null; git ls-files --others --exclude-standard)"; \
	if [ "$$before" != "$$after" ]; then echo "generate-go: go generate changed the files of the repository:" >&2; \
		git status --short >&2; exit 1; fi
audit-go: ## Scan every Go module for known vulnerabilities that its code reaches
	@$(GO) list -m -f '{{.Dir}}' | while IFS= read -r dir; do echo "govulncheck $$dir"; \
		$(GOVULNCHECK) -C "$$dir" $(GO_AUDIT_ARGS) $(GO_PATHS) || exit 1; done
check-go: lint-go test-go race-go audit-go ## Run the gate of Go
