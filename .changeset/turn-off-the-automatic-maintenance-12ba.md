---
"go.dokimi.dev/ergon/service": patch
---

Turn off the automatic maintenance of git in vcstest.Isolate, which a commit of git 2.48 and later starts in the background, so no process of git writes into the working tree of a test after the test ends.
