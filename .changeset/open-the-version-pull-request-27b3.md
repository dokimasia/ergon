---
"go.dokimi.dev/ergon": minor
"go.dokimi.dev/ergon/service": minor
---

Open the version pull request in version.yml when the run of ci.yml for a push to main passes, and publish in release.yml after ergon release ci verify finds a run of ci.yml that passed on the content of the commit, such as the run of the version pull request. ergon release ci verify replaces ergon release ci wait, so no job waits for another workflow. ergon release ci version skips a commit that is no longer the head of main. A GitHub App in the variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY opens the version pull request, so its checks run without an approval. Each push to main gets a concurrency group of its own in ci.yml.
