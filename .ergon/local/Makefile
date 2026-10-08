# Copyright Dokimasia B.V. 2026
# SPDX-License-Identifier: MIT

# The targets of ergon's own repository, which ergon init appends to the Makefile.

##@ Repository

.PHONY: update-baseline

update-baseline: ## Move the pins of the baselines of ergon's producers to their newest releases, with GITHUB_TOKEN set
	$(GO) run ./internal/cmd/update-baseline
