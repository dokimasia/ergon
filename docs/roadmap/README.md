# Roadmap

What we are building, in what order, and what has to be true first. Milestone numbers are permanent. The order is the order of this table, and it changes.

| Order | Milestone | Status | Depends on |
|---|---|---|---|
| 1 | 0007 License headers: `ergon license fix` and `check` on skywalking-eyes | Planned | none |
| 2 | 0000 Vocabulary, changeset files and the release planner | Planned | 0007 |
| 3 | 0001 TypeScript and JavaScript releases, checked against changesets 3.0.3 on the same repositories | Planned | 0000 |
| 4 | 0002 Go releases in one commit through the file proxy | Planned | 0000 |
| 5 | 0003 The CI flow: select-mode, version, pack and publish | Planned | 0001, 0002 |
| 6 | 0004 Rust releases | Planned | 0000 |
| 7 | 0005 Python releases | Planned | 0000 |
| 8 | 0006 Java and Kotlin releases through Gradle | Planned | 0000 |

The module layout, `go.work`, the depguard rules and CI are built and proven on the license command, because it is small and depends on no other milestone.

JavaScript comes before Go because changesets then acts as the reference implementation. The planner and the changelog are compared with changesets' output on the same changeset files before any other language depends on them.

## Deferred

| Item | Deferred | Blocks |
|---|---|---|
| The `go.dokimi.dev` vanity pages publish the subdirectory field for every module whose directory differs from its path suffix | 2026-09-24 | `go get` and `go install` of every ergon module except the root, and of techne's, eidos's and treesitter's nested modules |
