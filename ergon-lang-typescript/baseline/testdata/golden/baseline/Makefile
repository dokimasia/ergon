
# JavaScript and TypeScript
#
# The section js of .ergon.yaml sets the tools and the options of the targets that JavaScript and
# TypeScript share. One run overrides an option on the command line, such as
# make audit-js JS_AUDIT_SEVERITY=high.

# The tools of the shared targets, which ergon runs at the versions of the section js.
BIOME := $(ERGON) tool run js.biome --

# The options of the section js.
JS_PATHS ?= .
JS_AUDIT_SEVERITY ?= low

.PHONY: fmt-js lint-js audit-js
fmt: fmt-js
audit: audit-js

fmt-js: ## Format the JavaScript and TypeScript sources with Biome
	$(BIOME) check --write --linter-enabled=false $(JS_PATHS)
lint-js: ## Check the format and the lint of the JavaScript and TypeScript sources with biome.json
	$(BIOME) ci --error-on-warnings --skip=correctness/noNodejsModules $(JS_PATHS)
audit-js: ## Scan package-lock.json for known vulnerabilities
	npm audit --audit-level=$(JS_AUDIT_SEVERITY) --include=dev

# TypeScript
#
# The section typescript of .ergon.yaml sets the tools and the options of the TypeScript targets,
# and the steps of check-typescript. One run overrides an option on the command line, such as
# make test-typescript TYPESCRIPT_TEST_ARGS=test/unit.

# The tools of the TypeScript targets, which ergon runs at the versions of the section typescript.
TSC := $(ERGON) tool run typescript.typescript --

# The options of the section typescript.
TYPESCRIPT_TEST_ARGS ?=

.PHONY: lint-typescript test-typescript audit-typescript check-typescript
lint: lint-typescript
test: test-typescript
check: check-typescript

lint-typescript: lint-js ## Check the TypeScript sources with Biome, and with tsc in its strictest mode
	$(TSC) --noEmit --strict --noUncheckedIndexedAccess --noImplicitOverride --noImplicitReturns \
		--noFallthroughCasesInSwitch --noPropertyAccessFromIndexSignature --exactOptionalPropertyTypes
test-typescript: ## Run the TypeScript tests
	npm test $(if $(TYPESCRIPT_TEST_ARGS),-- $(TYPESCRIPT_TEST_ARGS))
audit-typescript: audit-js ## Scan the packages of TypeScript for known vulnerabilities
check-typescript: lint-typescript test-typescript audit-typescript ## Run the gate of TypeScript
