---
"go.dokimi.dev/ergon": minor
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon/lang/bash": none
"go.dokimi.dev/ergon/lang/csharp": none
"go.dokimi.dev/ergon/lang/go": none
"go.dokimi.dev/ergon/lang/java": none
"go.dokimi.dev/ergon/lang/javascript": none
"go.dokimi.dev/ergon/lang/kotlin": none
"go.dokimi.dev/ergon/lang/php": none
"go.dokimi.dev/ergon/lang/python": none
"go.dokimi.dev/ergon/lang/rust": none
"go.dokimi.dev/ergon/lang/terraform": none
"go.dokimi.dev/ergon/lang/typescript": none
---

Wait for the CI run of a commit before release.yml versions or publishes it. The job wait runs a new command for this, ergon release ci wait. The workflow release.yml no longer calls ci.yml, so each push to main runs the gate once.
