---
adr: 0029
title: ergon tool run builds golangci-lint with the module plugins of go.lint.plugins
status: Accepted
date: 2026-10-09
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0029: ergon tool run builds golangci-lint with the module plugins of go.lint.plugins

## Status

Accepted

## Context

golangci-lint runs a linter of another module only when its program contains the linter. The configuration of golangci-lint enables a linter and passes it settings, but it cannot add a linter to the program. golangci-lint has two plugin systems for such a linter:

- A module plugin is a Go package that registers a linter by name. `golangci-lint custom` clones golangci-lint at a version with git and adds each plugin with `go get`. It then builds a new program with the go command. It reads the version and the plugins from `.custom-gcl.yml`, `.custom-gcl.yaml` or `.custom-gcl.json` in its working directory.
- A Go plugin is a `.so` file that the program opens when it starts. Go supports plugins only on Linux, FreeBSD and macOS. The program and each plugin must be built with the same toolchain, build tags and flags.

assert-go publishes its analyzer as the module plugin `go.dokimi.dev/assert/lint/golangci`. Its package registers the linter `assertlint`. ergon does not run an assertlint tool of its own.

`ergon tool run go.golangci-lint` installs the golangci-lint of `go.tools.golangci-lint` into the tool cache of ergon with `go install`, once for each version of the go command. `fmt-go` and `lint-go` run golangci-lint this way. CI keeps the tool cache between runs, under a key that changes with `.ergon.yaml` and the lock. Without the cache, the first lint of each Go job of ergon's repository built golangci-lint from 203 modules in 57 s on Linux, 69 s on macOS and 104 s on Windows.

We measured `golangci-lint custom` v2.14.0 on 2026-10-09:

- It built golangci-lint with assertlint v0.1.0 from a `.custom-gcl.json` in 15.6 s, with warm caches of modules and builds. The program has 52 MB.
- A golangci-lint without the plugin refuses a configuration that enables it, with `build linters: plugin(assertlint): plugin "assertlint" not found`.
- With `AssertLint` in `linters.enable` and in `linters.settings.custom`, the program with the plugin refused the configuration with `unknown linters: 'AssertLint'`. The lowercase name passed.

The accepted design gives the step `lint` an option only for a tool without a configuration file.

## Decision

We will build the module plugins of `go.lint.plugins` into the golangci-lint that `ergon tool run` runs, because golangci-lint runs a module plugin only from a program that `golangci-lint custom` built with it.

- `go.lint.plugins` maps the name of a linter to the `<package>@<version>` of the Go package that registers it, such as `assertlint: go.dokimi.dev/assert/lint/golangci@v0.1.0`. The baseline has no plugin. A name starts with a lowercase letter, and its other characters are lowercase letters, digits, hyphens and underscores.
- The tag `plugins:"lint.plugins"` of `go.tools.golangci-lint` names the option of its plugins. `ergon tool run` installs golangci-lint with `go install` as before. When the option lists a plugin, it writes `.custom-gcl.json` into a temporary directory of the cache and runs `golangci-lint custom` there. It then renames the directory, with the program and its configuration, into the cache beside the program of `go install`. The cache keeps one program for each set of plugin packages, under the digest of its configuration.
- The managed `.golangci.yml` enables each linter of `go.lint.plugins` after the linters of the baseline, with the type `module` under `linters.settings.custom`. A repository adds the settings and the exclusions of a plugin in its local file.
- The step `lint` takes the option `go.lint.plugins` although golangci-lint has a configuration file, because the configuration cannot add a linter to the program.

## Alternatives Considered

### A managed `.custom-gcl.yml` and a target of the Makefile that builds the program

`ergon init` would render `.custom-gcl.yml` into the root of each repository with plugins. `lint-go` would build the program into the repository before its first run. It lost because the program would be outside the tool cache, so every job of CI would build golangci-lint again. Such a build took 57 to 104 s per job without the cache. The option also adds a managed file to the root of the repository.

### Go plugins

The configuration would give each plugin the type `goplugin` and the path of its `.so` file. Go plugins lost because Go does not support them on Windows, and CI runs the gate of Go on Windows. Each plugin would also need the same toolchain, build tags and flags as the golangci-lint that opens it.

### A tool of assertlint beside golangci-lint

`lint-go` would run assertlint as a tool of `go.tools` after golangci-lint, as it runs ergon-go-vet. It lost because golangci-lint applies its exclusions and its `//nolint` directives to the findings of a plugin, as it does for every other linter. A second tool would need exclusions of its own.

## Consequences

**Positive:**

- One option builds a plugin into golangci-lint and enables its linter in every module of the repository.
- CI builds the program once for each change of `.ergon.yaml` or the lock, because the program is in the tool cache.

**Negative:**

- The first run after a change of the plugins, of golangci-lint or of the go command builds golangci-lint. The build needs git, and access to github.com and to the module proxy.
- A golangci-lint that ergon did not build, such as the one of an editor, refuses the configuration of a repository with plugins.
- `service/tool` writes the configuration file of `golangci-lint custom` and runs that command, so it depends on the interface of one command of one tool.
- The cache keeps a second program of golangci-lint for each set of plugins.
- The baseline update resolves no plugin, because the baseline has none. A repository raises the version of a plugin in its `.ergon.yaml` itself.

**Neutral:**

- A repository without plugins runs the golangci-lint of `go install`, as before.

## References

| What | Where |
|---|---|
| The design of the tools and their cache | RFC-0004, Tools |
| The module plugins of golangci-lint | https://golangci-lint.run/docs/plugins/module-plugins/ |
| The build of `golangci-lint custom` | `pkg/commands/internal/builder.go` and `configuration.go` of golangci-lint v2.14.0 |
| The limits of Go plugins | https://pkg.go.dev/plugin |
| The module plugin of assertlint | https://github.com/dokimasia/assert-go/tree/main/lint/golangci |
