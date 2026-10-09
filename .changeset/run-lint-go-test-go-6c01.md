---
"go.dokimi.dev/ergon": patch
"go.dokimi.dev/ergon/lang/go": patch
---

Run lint-go, test-go, race-go and audit-go in every module, and fail after the last module when one of them failed. The job check-go of ci.yml runs make --keep-going check-go, so a failed step does not skip the steps after it. The managed .golangci.yml does not run wrapcheck on test files and on the files of generators.
