
##@ JavaScript and TypeScript
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

##@ JavaScript
#
# The section javascript of .ergon.yaml sets the options of the JavaScript targets and the steps of
# check-javascript. One run overrides an option on the command line, such as
# make test-javascript JAVASCRIPT_TEST_ARGS=test/unit.

# The options of the section javascript.
JAVASCRIPT_TEST_ARGS ?=

.PHONY: lint-javascript test-javascript audit-javascript check-javascript
lint: lint-javascript
test: test-javascript
check: check-javascript

lint-javascript: lint-js ## Check the JavaScript sources with Biome
test-javascript: ## Run the JavaScript tests
	npm test $(if $(JAVASCRIPT_TEST_ARGS),-- $(JAVASCRIPT_TEST_ARGS))
audit-javascript: audit-js ## Scan the packages of JavaScript for known vulnerabilities
check-javascript: lint-javascript test-javascript audit-javascript ## Run the gate of JavaScript
