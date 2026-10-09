<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFCs

Proposals, and the argument that produced them. An RFC is written to be argued with. Numbers are never reused and never renumbered, including for proposals that are withdrawn.

| RFC | Title | Status |
|---|---|---|
| [0001](0001-module-boundaries.md) | Module boundaries | Accepted |
| [0002](0002-release.md) | Release | Accepted, superseded in part by RFC-0007, ADR-0016 and ADR-0018 |
| [0003](0003-license-headers.md) | Licenses | Accepted |
| [0004](0004-repository-initialization.md) | Repository initialization | Accepted, superseded in part by RFC-0005, RFC-0006, RFC-0007, ADR-0016, ADR-0017 and ADR-0018 |
| [0005](0005-baseline-updates.md) | Baseline updates | Accepted |
| [0006](0006-configurable-hooks.md) | Configurable hooks | Accepted |
| [0007](0007-binary-releases.md) | Binary releases | Accepted, superseded in part by ADR-0022 and ADR-0023 |

RFC-0001 fixes which module each part of a command belongs in, and why each language is a module of its own. RFC-0002 specifies `ergon release`: the changeset files, the planner, the version pull request, and publishing for each language. RFC-0003 specifies the licenses that ergon supports, their texts, and `ergon license`, which keeps the header of every file. RFC-0004 specifies `ergon init`: the producers of the files of a repository, their options, the jobs of CI, the tools of the gate, the lock, and the commands that keep a repository at the baseline. RFC-0005 specifies how the pins of the baseline update: the resolvers of the registries, the weekly update in ergon's repository, and `ergon init upgrade`. RFC-0006 lets a repository choose the targets of the Makefile that its hooks run before each commit and before each push. RFC-0007 specifies how each release of a Go module attaches the signed archives, packages and casks of the commands of the module.
