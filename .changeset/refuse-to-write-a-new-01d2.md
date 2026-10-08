---
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon": minor
---

Refuse to write a new lock, or a lock that a release wrote, in a build of ergon without a release, whose version dev no release has. Such a build keeps a lock that such a build wrote.
