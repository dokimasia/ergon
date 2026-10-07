
# Terraform
#
# The section terraform of .ergon.yaml sets the tools and the options of the Terraform targets, and
# the steps of check-terraform. One run overrides an option on the command line, such as
# make test-terraform TERRAFORM_TEST_ARGS=-verbose.

# The tools of the Terraform targets, which ergon installs at the versions of the section
# terraform. checkov runs on Python 3.13.
TFLINT := $(ERGON) tool run terraform.tflint --
CHECKOV := UV_PYTHON=3.13 $(ERGON) tool run terraform.checkov --

# The options of the section terraform.
TERRAFORM_PATHS ?= .
TERRAFORM_TEST_ARGS ?=
TERRAFORM_AUDIT_IGNORE ?=

.PHONY: fmt-terraform lint-terraform test-terraform audit-terraform check-terraform
fmt: fmt-terraform
lint: lint-terraform
test: test-terraform
audit: audit-terraform
check: check-terraform

fmt-terraform: ## Format the Terraform sources
	terraform fmt -recursive $(TERRAFORM_PATHS)
lint-terraform: ## Check the format of the Terraform sources, and lint every module with .tflint.hcl
	terraform fmt -check -recursive $(TERRAFORM_PATHS)
	@for path in $(TERRAFORM_PATHS); do echo "tflint $$path"; \
		$(TFLINT) --chdir="$$path" --recursive --config="$(CURDIR)/.tflint.hcl" || exit 1; done
test-terraform: ## Run the Terraform tests
	terraform init -backend=false -input=false
	terraform test $(TERRAFORM_TEST_ARGS)
audit-terraform: ## Scan the Terraform configuration for security misconfigurations
	@set --; for path in $(TERRAFORM_PATHS); do set -- "$$@" --directory "$$path"; done; \
	for id in $(TERRAFORM_AUDIT_IGNORE); do set -- "$$@" --skip-check "$$id"; done; \
	echo "checkov $$*"; \
	CKV_PARSE_ERROR_FAIL=true $(CHECKOV) "$$@" --framework terraform --quiet --compact --skip-download
check-terraform: lint-terraform test-terraform audit-terraform ## Run the gate of Terraform
