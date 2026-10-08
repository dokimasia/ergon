---
"go.dokimi.dev/ergon": minor
"go.dokimi.dev/ergon/core": minor
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon/lang/go": minor
---

Render nightly.yml, which runs the long steps of each language on a schedule, each step in a job of its own. The key nightly of the section go maps fuzz, bench and mutate to the limit of each job in minutes, and the key nightly.schedule of the section github states when the workflow runs.
