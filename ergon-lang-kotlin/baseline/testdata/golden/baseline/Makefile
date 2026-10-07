
# Java and Kotlin
#
# The section jvm of .ergon.yaml sets the tools of the targets that Java and Kotlin share.

# The tools of the shared targets, which ergon runs at the versions of the section jvm.
OSV_SCANNER := $(ERGON) tool run jvm.osv-scanner --

.PHONY: audit-jvm
audit: audit-jvm

audit-jvm: ## Scan the Gradle lockfiles for known vulnerabilities
	@lockfiles="$$(git ls-files --cached --others --exclude-standard -- ':(glob)**/gradle.lockfile' ':(glob)**/buildscript-gradle.lockfile')"; \
	if [ -z "$$lockfiles" ]; then echo "audit-jvm: no gradle.lockfile, so enable dependency locking in the Gradle build"; exit 1; fi; \
	set --; for lockfile in $$lockfiles; do set -- "$$@" "--lockfile=$$lockfile"; done; \
	$(OSV_SCANNER) scan source "$$@"

# Kotlin
#
# The section kotlin of .ergon.yaml sets the tools and the options of the Kotlin targets, and the
# steps of check-kotlin. One run overrides an option on the command line, such as
# make test-kotlin KOTLIN_TEST_ARGS=--info.

# The tools of the Kotlin targets, which ergon runs at the versions of the section kotlin. ergon
# checks the jar of ktlint against the checksum that Maven Central publishes, and runs it with java.
KTLINT := $(ERGON) tool run kotlin.ktlint --

# The options of the section kotlin.
KOTLIN_TEST_ARGS ?=

.PHONY: fmt-kotlin lint-kotlin test-kotlin audit-kotlin check-kotlin
fmt: fmt-kotlin
lint: lint-kotlin
test: test-kotlin
check: check-kotlin

fmt-kotlin: ## Format the Kotlin sources with ktlint
	git ls-files -z --cached --others --exclude-standard -- '*.kt' '*.kts' | \
		$(KTLINT) --relative --format --patterns-from-stdin=
lint-kotlin: ## Check the Kotlin sources with ktlint, and the build's own checks
	git ls-files -z --cached --others --exclude-standard -- '*.kt' '*.kts' | \
		$(KTLINT) --relative --patterns-from-stdin=
	./gradlew check -x test
test-kotlin: ## Run the Kotlin tests
	./gradlew test $(KOTLIN_TEST_ARGS)
audit-kotlin: audit-jvm ## Scan the Gradle lockfiles of Kotlin for known vulnerabilities
check-kotlin: lint-kotlin test-kotlin audit-kotlin ## Run the gate of Kotlin
