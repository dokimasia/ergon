---
"go.dokimi.dev/ergon": minor
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon/core": minor
"go.dokimi.dev/ergon/lang/csharp": none
"go.dokimi.dev/ergon/lang/go": none
"go.dokimi.dev/ergon/lang/javascript": none
"go.dokimi.dev/ergon/lang/kotlin": none
"go.dokimi.dev/ergon/lang/php": none
"go.dokimi.dev/ergon/lang/python": none
"go.dokimi.dev/ergon/lang/rust": none
"go.dokimi.dev/ergon/lang/typescript": none
---

Add ergon init upgrade and ergon init ci upgrade, which move a repository to the baseline of the newest release of ergon. The job of baseline.yml opens the pull request of the upgrade weekly, in place of an issue. ergon refuses a lock that a newer release of ergon wrote, and a local dependabot.yml that updates github-actions or pre-commit.
