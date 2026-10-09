---
"go.dokimi.dev/ergon": patch
"go.dokimi.dev/ergon/core": patch
"go.dokimi.dev/ergon/service": patch
"go.dokimi.dev/ergon/lang/bash": patch
"go.dokimi.dev/ergon/lang/csharp": patch
"go.dokimi.dev/ergon/lang/go": patch
"go.dokimi.dev/ergon/lang/java": patch
"go.dokimi.dev/ergon/lang/javascript": patch
"go.dokimi.dev/ergon/lang/kotlin": patch
"go.dokimi.dev/ergon/lang/php": patch
"go.dokimi.dev/ergon/lang/python": patch
"go.dokimi.dev/ergon/lang/rust": patch
"go.dokimi.dev/ergon/lang/terraform": patch
"go.dokimi.dev/ergon/lang/typescript": patch
---

Require Go 1.27.2, whose standard library fixes the nine vulnerabilities that govulncheck reports for Go 1.27.1, such as GO-2026-6604 of os.Root on Windows.
