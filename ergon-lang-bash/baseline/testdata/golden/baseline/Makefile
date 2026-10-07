
# Bash
#
# The section bash of .ergon.yaml sets the tools and the paths of the Bash targets, and the steps of
# check-bash.

# The tools of the Bash targets, which ergon installs at the versions of the section bash.
SHELLCHECK := $(ERGON) tool run bash.shellcheck --

# The options of the section bash: the pathspecs of git of the scripts.
BASH_PATHS ?= '*.sh' '*.bash'

.PHONY: lint-bash check-bash
lint: lint-bash
check: check-bash

lint-bash: ## Check the Bash sources with shellcheck
	git ls-files -z --cached --others --exclude-standard -- $(BASH_PATHS) | xargs -0 -r $(SHELLCHECK)
check-bash: lint-bash ## Run the gate of Bash
