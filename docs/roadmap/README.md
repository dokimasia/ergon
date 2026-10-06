# Roadmap

What we are building, in what order, and what has to be true first. Milestone numbers are permanent. The order is the order of this table, and it changes.

| Order | Milestone | Status | Depends on |
|---|---|---|---|
| 1 | 0008 Repository initialization: `ergon init` with the common files, the GitHub files and the fragments of the eleven languages | Built | none |
| 2 | 0007 License headers: `ergon license fix` and `check` on skywalking-eyes, and the job `license` of the workflow of the gate | Planned | none |
| 3 | 0000 Vocabulary, changeset files and the release planner | Planned | 0007 |
| 4 | 0001 TypeScript and JavaScript releases, checked against changesets 3.0.3 on the same repositories | Planned | 0000 |
| 5 | 0002 Go releases in one commit through the file proxy | Planned | 0000 |
| 6 | 0003 The CI flow: select-mode, version, pack and publish, the workflow `release.yml` and the job `changeset` of ergon init | Planned | 0001, 0002 |
| 7 | 0004 Rust releases | Planned | 0000 |
| 8 | 0005 Python releases | Planned | 0000 |
| 9 | 0006 Java and Kotlin releases through Gradle | Planned | 0000 |

The module layout, `go.work` and the depguard rules are built and proven on `ergon init`.

The release of ergon publishes `ergon_<version>_<os>_<arch>.tar.gz` for `linux`, `darwin` and `windows` on `amd64` and `arm64`, and `checksums.txt`, which the action `setup-ergon` of ergon init downloads. ergon has no release yet, so the jobs of a repository that run ergon fail until its first one.

JavaScript comes before Go because changesets then acts as the reference implementation. The planner and the changelog are compared with changesets' output on the same changeset files before any other language depends on them.

## Deferred

| Item | Deferred | Blocks |
|---|---|---|
| The `go.dokimi.dev` vanity pages publish the subdirectory field for every module whose directory differs from its path suffix | 2026-09-24 | `go get` and `go install` of every ergon module except the root, and of techne's, eidos's and treesitter's nested modules |
