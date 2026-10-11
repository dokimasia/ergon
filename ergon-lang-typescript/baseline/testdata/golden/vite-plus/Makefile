
##@ JavaScript and TypeScript
#
# The section js of .ergon.yaml sets the package manager, the toolkit, the tools and the options of
# the targets that JavaScript and TypeScript share. One run overrides an option on the command line,
# such as make audit-js JS_AUDIT_SEVERITY=high.

# The vp of the packages of the repository, which the package manager runs.
VP := pnpm exec vp

# The command that runs the tests of JavaScript and TypeScript, before the arguments of a run.
JS_TEST := $(VP) test --run

# The options of the section js.
JS_AUDIT_SEVERITY ?= low

.PHONY: fmt-js lint-js audit-js
fmt: fmt-js
audit: audit-js

fmt-js: ## Format the sources with vp fmt and the fmt block of vite.config.ts
	$(VP) fmt
lint-js: ## Check the format, the lint and the types of the sources with vp check
	$(VP) check
audit-js: ## Scan pnpm-lock.yaml for known vulnerabilities
	pnpm audit --audit-level=$(JS_AUDIT_SEVERITY)

##@ TypeScript
#
# The section typescript of .ergon.yaml sets the tools and the options of the TypeScript targets,
# and the steps of check-typescript. One run overrides an option on the command line, such as
# make test-typescript TYPESCRIPT_TEST_ARGS=test/unit.

# The options of the section typescript.
TYPESCRIPT_TEST_ARGS ?=

.PHONY: lint-typescript test-typescript audit-typescript check-typescript
lint: lint-typescript
test: test-typescript
check: check-typescript

lint-typescript: lint-js ## Check the TypeScript sources with vp check
test-typescript: ## Run the TypeScript tests
	$(JS_TEST)$(if $(TYPESCRIPT_TEST_ARGS), $(TYPESCRIPT_TEST_ARGS))
audit-typescript: audit-js ## Scan the packages of TypeScript for known vulnerabilities
check-typescript: lint-typescript test-typescript audit-typescript ## Run the gate of TypeScript
