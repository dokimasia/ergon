
##@ Terraform
#
# The section terraform of .ergon.yaml sets the tools and the options of the Terraform targets, and
# the steps of check-terraform. One run overrides an option on the command line, such as
# make test-terraform TERRAFORM_TEST_ARGS=-verbose.

# The terraform command.
TERRAFORM ?= terraform

# The tools of the Terraform targets, which ergon installs at the versions of the section
# terraform. checkov runs on Python 3.13.
TFLINT := $(ERGON) tool run terraform.tflint --
TERRAFORM_DOCS := $(ERGON) tool run terraform.terraform-docs --
CHECKOV := UV_PYTHON=3.13 $(ERGON) tool run terraform.checkov --

# The plugin directory of tflint: the links to the rulesets of the section, and the plugins that
# tflint --init installs for the local .tflint.hcl.
TFLINT_PLUGIN_DIR := $(CURDIR)/.tflint.d/plugins

# The options of the section terraform.
TERRAFORM_PATHS ?= .
TERRAFORM_LINT_RULESETS ?=
TERRAFORM_TEST_ARGS ?=
TERRAFORM_AUDIT_IGNORE ?=

# terraform-directories is the program of awk that reads the files under the paths of the section,
# one on each line, and writes a line for each directory of Terraform: test and the directory for a
# directory with test files in it or in its tests directory, and validate and the directory for
# each other directory with a configuration.
terraform-directories = { dir = $$0; if (!sub(/\/[^\/]*$$/, "", dir)) dir = "." } \
	/\.tftest\.(hcl|json)$$/ { if (dir == "tests") dir = "."; else sub(/\/tests$$/, "", dir); \
	tested[dir] = 1; next } \
	/\.tf(\.json)?$$/ { configured[dir] = 1 } \
	END { for (dir in tested) print "test", dir; for (dir in configured) if (!(dir in tested)) print "validate", dir }

.PHONY: fmt-terraform lint-terraform test-terraform audit-terraform check-terraform
fmt: fmt-terraform
lint: lint-terraform
test: test-terraform
audit: audit-terraform
check: check-terraform

fmt-terraform: ## Format the Terraform sources
	$(TERRAFORM) fmt -recursive $(TERRAFORM_PATHS)
lint-terraform: ## Check the format of the Terraform sources, and lint every module with .tflint.hcl
	$(TERRAFORM) fmt -check -recursive $(TERRAFORM_PATHS)
	@mkdir -p "$(TFLINT_PLUGIN_DIR)" && for ruleset in $(TERRAFORM_LINT_RULESETS); do \
		program="$$($(ERGON) tool path terraform.tflint-ruleset-$$ruleset)" && \
		ln -sf "$$program" "$(TFLINT_PLUGIN_DIR)/" || exit 1; done
	TFLINT_PLUGIN_DIR="$(TFLINT_PLUGIN_DIR)" $(TFLINT) --init --config="$(CURDIR)/.tflint.hcl"
	@for path in $(TERRAFORM_PATHS); do echo "tflint $$path"; \
		TFLINT_PLUGIN_DIR="$(TFLINT_PLUGIN_DIR)" $(TFLINT) --chdir="$$path" --recursive \
			--config="$(CURDIR)/.tflint.hcl" || exit 1; done
test-terraform: ## Run the tests of each Terraform module, and validate every other configuration
	@files="$$(git -c core.quotePath=false ls-files --cached --others --exclude-standard -- $(TERRAFORM_PATHS))" \
		|| exit 1; \
	dirs="$$(printf '%s\n' "$$files" | awk '$(terraform-directories)')"; \
	if [ -z "$$dirs" ]; then echo "no configuration of Terraform in $(TERRAFORM_PATHS)"; exit 0; fi; \
	printf '%s\n' "$$dirs" | { status=0; while read -r step dir; do echo "terraform $$step $$dir"; \
		$(TERRAFORM) -chdir="$$dir" init -backend=false -input=false && \
		if [ "$$step" = test ]; then $(TERRAFORM) -chdir="$$dir" test $(TERRAFORM_TEST_ARGS); \
		else $(TERRAFORM) -chdir="$$dir" validate; fi || status=1; done; exit $$status; }
audit-terraform: ## Scan the Terraform configuration for security misconfigurations
	@set --; for path in $(TERRAFORM_PATHS); do set -- "$$@" --directory "$$path"; done; \
	for id in $(TERRAFORM_AUDIT_IGNORE); do set -- "$$@" --skip-check "$$id"; done; \
	echo "checkov $$*"; \
	CKV_PARSE_ERROR_FAIL=true $(CHECKOV) "$$@" --framework terraform --quiet --compact --skip-download
check-terraform: lint-terraform test-terraform audit-terraform ## Run the gate of Terraform
