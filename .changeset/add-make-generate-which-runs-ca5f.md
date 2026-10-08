---
"go.dokimi.dev/ergon/core": minor
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon/lang/bash": minor
"go.dokimi.dev/ergon/lang/csharp": minor
"go.dokimi.dev/ergon/lang/go": minor
"go.dokimi.dev/ergon/lang/java": minor
"go.dokimi.dev/ergon/lang/javascript": minor
"go.dokimi.dev/ergon/lang/kotlin": minor
"go.dokimi.dev/ergon/lang/php": minor
"go.dokimi.dev/ergon/lang/python": minor
"go.dokimi.dev/ergon/lang/rust": minor
"go.dokimi.dev/ergon/lang/terraform": minor
"go.dokimi.dev/ergon/lang/typescript": minor
---

Add make generate, which runs the generators of every language, and the options generate.command and generate.args to every language section of .ergon.yaml. generate-<language> runs the generators, and verify-generate-<language>, the step generate of check, fails when they change a file. make help lists the targets in groups: Common, each language, and the targets of the repository.
