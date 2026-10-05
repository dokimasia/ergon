---
adr: 0001
title: One module per language
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: ADR-0008, in part
rfc: RFC-0001
---

# ADR-0001: One module per language

## Status

Accepted

## Context

Every ergon command has a part that differs per language and a part that does not. `release` rewrites five manifest formats in five ways and plans the release the same way for all of them. Test runners, coverage parsers and lint task lists follow the same split.

The per-language part brings its own dependencies and its own toolchain. Go needs `golang.org/x/mod`. Rust and Python need a TOML parser that reports byte ranges. Each language's tests need that language's toolchain installed: `go`, `cargo`, `uv`, Node with npm, pnpm and bun, or a JDK with Gradle.

A single module is simpler to release and to install. It also puts every language's dependencies in one `go.sum` and makes `go test ./...` need every toolchain at once.

## Decision

We will give each language its own Go module, `ergon-lang-<language>`, and put the code that two or more languages share in `ergon-lang`, because every command is implemented per language and each language brings dependencies and a toolchain that the others do not need.

The JavaScript and JVM toolchains are the packages `ergon-lang/js` and `ergon-lang/jvm`. Each serves two languages, and neither has a third-party dependency to contain.

## Alternatives Considered

### One module, with the layers as packages

depguard enforces the package positions without module boundaries. `go install` then works without pinning sibling versions, and each release has one version. It lost because ergon adds commands that are each implemented per language. One module would collect every language's dependencies and require every toolchain for `go test ./...`.

### One module per command

`ergon-release` would contain the planner and a package per language, and each later command would get a module of its own. It lost because Go's `go.mod` handling would appear in every command module that serves Go. Removing a language would also touch every command module.

### A module for each of the JavaScript and JVM toolchains

`ergon-lang-js` and `ergon-lang-jvm` would sit beside `ergon-lang-go`. It lost because neither toolchain has a third-party dependency. Code that serves two languages belongs in `ergon-lang`.

## Consequences

**Positive:**

- Removing a language is one directory, one `go.work` line and one `Register` call.
- `go test ./...` in a language's directory needs only that language's toolchain, so the CI matrix gets one toolchain per row.
- A language's third-party dependencies appear only in its own `go.sum`.

**Negative:**

- Seven `go.mod` files, and one more for each language added. Each needs its own tidy, tags and changelog.
- A change to a role interface edits `ergon-core`, `ergon-service` and every language module that implements the role, in one commit.
- `go.work` hides a missing `require`, so depguard is the only check on the dependency direction inside the workspace.
- TypeScript, JavaScript, Java and Kotlin have no module of their own. Code specific to one of them needs a new module when a command first requires it.

**Neutral:**

- The directories follow techne's names, so `go.dokimi.dev/ergon/lang/go` is in `ergon-lang-go/`.
