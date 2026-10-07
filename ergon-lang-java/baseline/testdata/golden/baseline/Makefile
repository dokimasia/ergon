
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

# Java
#
# The section java of .ergon.yaml sets the tools and the options of the Java targets, and the steps
# of check-java. One run overrides an option on the command line, such as
# make test-java JAVA_TEST_ARGS=--info.

# The options of the section java.
JAVA_TEST_ARGS ?=

.PHONY: lint-java test-java audit-java check-java
lint: lint-java
test: test-java
check: check-java

lint-java: ## Check the Java sources with PMD, and with javac and javadoc warnings as errors
	./gradlew --init-script gradle/ergon-java.init.gradle.kts check javadoc -x test
test-java: ## Run the Java tests
	./gradlew test $(JAVA_TEST_ARGS)
audit-java: audit-jvm ## Scan the Gradle lockfiles of Java for known vulnerabilities
check-java: lint-java test-java audit-java ## Run the gate of Java
