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

Skip the jobs of a run of ci.yml when a passed run covers its content: a push whose content passed, and a version commit whose parent passed. The new first job skip runs ergon release ci skip, ergon release ci version marks the commit of the version pull request with the status ergon/version, and the job result is the one check that a branch requires.
