
##@ Spec
#
# The section spec of .ergon.yaml sets the roles of the repository and the release of spec. spec
# reads its own configuration from .spec.yaml.

# The program spec, which ergon installs at the version of the section spec.
SPEC := $(ERGON) tool run spec.spec --

.PHONY: check-spec
check: check-spec

check-spec: ## Check the copies of the definitions against their manifests with spec
	$(SPEC) check

.PHONY: generate-spec verify-generate-spec
generate: generate-spec
check: verify-generate-spec

generate-spec: ## Write the manifest of the definition with spec
	$(SPEC) manifest
verify-generate-spec: ## Fail when the manifest of the definition differs from its files
	$(SPEC) manifest --check
