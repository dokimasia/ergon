<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Roadmap

What we are building, in what order, and what has to be true first. Milestone numbers are permanent. The order is the order of this table, and it changes.

| Order | Milestone | Status | Depends on |
|---|---|---|---|
| 1 | 0008 Repository initialization: `ergon init` with the common files, the GitHub files and the fragments of the eleven languages | Built | none |
| 2 | 0007 License headers: `ergon license fix` and `check` on skywalking-eyes, and the job `license` of the workflow of the gate | Built | none |
| 3 | 0000 Vocabulary, changeset files and the release planner | Built | 0007 |
| 4 | 0002 Go releases in one commit through the file proxy | Built | 0000 |
| 5 | 0003 The CI flow: select-mode, version, pack and publish, the workflow `release.yml` and the job `changeset` of ergon init | Built, except the recovery of `go.sum` in select-mode that the failure table of RFC-0002 states | 0002 |
| 6 | 0001 TypeScript and JavaScript releases, checked against changesets 3.0.3 on the same repositories | Planned | 0003 |
| 7 | 0004 Rust releases | Planned | 0003 |
| 8 | 0005 Python releases | Planned | 0003 |
| 9 | 0006 Java and Kotlin releases through Gradle | Planned | 0003 |

The module layout, `go.work` and the depguard rules are built and proven on `ergon init`.

Each release of the root module publishes `ergon_<version>_<os>_<arch>.tar.gz` for `linux`, `darwin` and `windows` on `amd64` and `arm64`, and `checksums.txt`, which the action `setup-ergon` of ergon init downloads. The workflow `binaries.yml` of ergon's repository builds them with GoReleaser.

Go comes before JavaScript, because ergon releases its own modules through the flow of Go. Milestone 0001 compares the plans and the changelogs of npm with changesets' output on the same changeset files.

## Deferred

None.
