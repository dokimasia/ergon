
##@ GitHub
#
# The section github of .ergon.yaml sets the tools of the GitHub targets.

# The tools of the GitHub targets, which ergon installs at the versions of the section github.
ACTIONLINT := $(ERGON) tool run github.actionlint --

.PHONY: lint-github
lint: lint-github
check: lint-github

# actionlint runs shellcheck and pyflakes on the scripts of the steps, and the flags turn both off.
# ergon pins neither program, so actionlint would run the versions that the machine has.
lint-github: ## Lint the workflows with actionlint and the configuration of .github/actionlint.yaml
	$(ACTIONLINT) -shellcheck= -pyflakes=
