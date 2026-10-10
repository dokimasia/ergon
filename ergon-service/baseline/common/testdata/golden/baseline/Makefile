# Managed by ergon init. Add repository settings to .ergon/local/Makefile and run ergon init sync.
#
# make check is the gate of the repository. Each language adds its own fmt, lint, test, generate,
# audit and check targets as prerequisites of the targets below, and CI runs make check-<language>
# per language. Every target runs its tools through ergon tool run, which installs the version that
# .ergon.yaml names. make help lists the targets in groups. A line ##@ <name> starts a group: the
# common targets, then the targets of each language, and the targets of .ergon/local/Makefile after
# a line of its own.

.DEFAULT_GOAL := check

# The command that runs ergon for the tools of the targets. The key ergon of the section common of
# .ergon.yaml sets it.
ERGON ?= ergon

# verify-generated runs the command of the variable $(1) and fails when the run changes a file of
# the repository, with the target $(2) that updates the files in its message. It compares the
# changes of the working tree before and after the run, so it also runs on a tree with uncommitted
# changes.
verify-generated = before="$$(git diff HEAD --binary 2>/dev/null; git ls-files --others --exclude-standard)"; \
	$($(1)) || exit 1; \
	after="$$(git diff HEAD --binary 2>/dev/null; git ls-files --others --exclude-standard)"; \
	if [ "$$before" != "$$after" ]; then \
	echo "$(2): the generated files are out of date, so run make $(2) and commit the result:" >&2; \
	git status --short >&2; exit 1; fi

.PHONY: help fmt lint test generate audit check

##@ Common

help: ## List the targets in groups
	@awk 'BEGIN {FS = ":.*## "} /^##@ / {group = substr($$0, 5); next} /^[a-z][a-z-]*:.*## / {if (group != shown) {printf "%s%s\n", (shown == "" ? "" : "\n"), group; shown = group} printf "  %-28s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format the sources of every language
lint: ## Lint the sources of every language
test: ## Run the tests of every language
generate: ## Run the generators of every language
audit: ## Run the vulnerability scan of every language
check: ## Run the gate of every language
