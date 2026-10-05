# Package layout

Where a file goes, and what each package may import. The module boundaries and the rules between modules are argued in RFC-0001. This document covers the packages inside them.

## Two rules place every file

**Each command is split in two.** The part that is the same for every language is in `ergon-service`. The part that differs is in a language module, or in `ergon-lang` when two languages share it. `core` declares the roles that connect the two parts.

**Machinery is placed by reuse.** A package belongs in `ergon-lang` when more than one language uses it, and in a language module when only one does. Rust and Python both read TOML manifests, so TOML editing is in `lang/manifest`. `go.mod` handling serves Go alone, so it is in `ergon-lang-go`.

## Dependency position

Every package declares its position in its `doc.go`, under a `# Dependency position` heading that names what it imports. The tables in this document collect the same information. A reader meets the `doc.go` copy first, so a change to a package's imports updates its `doc.go` in the same commit.

Positions are scoped to a module and run from 0. A package imports packages at lower positions. Packages at the same position import each other only where a table lists the import. Across modules, the rules in RFC-0001 apply, and depguard enforces them.

## ergon-core

The vocabulary that every command uses, and the roles that toolchains and languages implement. Nothing here consumes a role.

### Position 0: vocabulary

| Package | Contains | Imports |
|---|---|---|
| `version` | `Version`, `Bump`, parsing and bumping of SemVer 2.0.0 | stdlib |
| `changeset` | `Changeset`, `Parse`, `Format` for `.changeset/*.md` | stdlib, `version` |
| `workspace` | `Toolchain`, `Language`, `Package`, `Dependency`, `Kind` | `version` |

A type belongs here when a role signature names it. A type that does something, or that only one command reads, belongs in that command's package.

### Position 1: toolchains, languages and roles

| Package | Contains | Imports |
|---|---|---|
| `language` | `Toolchain`, `Declaration`, `Catalog`, `RegisterToolchain`, `Register`, and the role interfaces with one file per command: `release.go` contains `Versioner`, `Packer`, `Publisher` and `Edit`, and `init.go` contains `Initializer`, `File`, `Class` and `Answers` | position 0 |

The catalog has two kinds of entry. Toolchains discover packages and implement the release roles. Each language declares its toolchain and implements `Initializer`. The Java module also implements `Initializer` for its toolchain, to render the Gradle build that Java and Kotlin share. Roles are separate interfaces, so a toolchain or a language declines a command by not implementing its role. The catalog selects by type assertion, and each implementation asserts its roles at compile time.

## ergon-service

The language-neutral side of each command. It imports `core` and nothing else in the repository.

| Package | Contains | Imports |
|---|---|---|
| `release` | The planner, the changelog writer, the publish plan, tagging, and the `Forge` interface it calls | `core/*`, `vcs`, `workspace` |
| `license` | `Config`, `Check`, `Fix`, the comment-style overrides and the removal of outdated header blocks | `core/*`, `vcs`, skywalking-eyes `pkg/header`, `pkg/comments` and `pkg/logger` |
| `baseline` | The `init` command: composition, rendering, the merge of local files, the lock, and `New`, `Add`, `Remove`, `Check` and `Sync` | `core/*`, `baseline/common`, `baseline/github` |
| `baseline/common` | The templates of the common files | stdlib |
| `baseline/github` | The templates of the GitHub files, and the CI rules that every workflow follows | stdlib |
| `vcs` | git: the diff since a ref, tags, the snapshot commit, the atomic push | stdlib |
| `forge` | The GitHub client: signed commits through `createCommitOnBranch`, branch and tag refs, pull requests, releases, and the pull request behind a commit | stdlib, `core/workspace` |
| `workspace` | Which toolchains and languages are active in a repository, and which package a changed file belongs to | `core/language`, `core/workspace`, `vcs` |

`vcs` is the only package in this module that runs git, and `forge` is the only package that calls the GitHub API. `forge` satisfies `release.Forge` without importing `release`, and no other package in the module imports `forge`. `license` is the only package in the repository that imports skywalking-eyes.

## ergon-lang

Machinery that two or more languages use. It imports `core` and nothing else in the repository.

| Package | Contains | Imports |
|---|---|---|
| `manifest` | Byte-range edits for JSON, TOML, XML and properties files that keep formatting and comments | stdlib, go-toml v2 |
| `command` | The subprocess runner | stdlib |
| `conformance` | One suite per role. Every toolchain and language runs it against its fixtures | `core/language`, `core/workspace` |

A language module imports `manifest`, `command` and `conformance`. The only imports between language modules are two: `ergon-lang-typescript` imports the root package of `ergon-lang-javascript`, and `ergon-lang-kotlin` imports the root package of `ergon-lang-java`, for the name of their toolchain.

## A language module

```text
ergon-lang-go/
  doc.go        package golang
  language.go   Language, Toolchain and Register
  workspace/    go.work and go.mod discovery, for every command
  release/      Versioner: require rewrites, go.sum through the file proxy, tag prefixes
  baseline/     Initializer: templates, gate tools and fragments
```

| Module | Root package | Toolchain | Packages |
|---|---|---|---|
| `ergon-lang-csharp` | `csharp` | `csharp` | `workspace`, `release`, `baseline` |
| `ergon-lang-java` | `java` | `jvm` | `workspace`, `release`, `baseline` |
| `ergon-lang-kotlin` | `kotlin` | `jvm`, from `ergon-lang-java` | `baseline` |
| `ergon-lang-php` | `php` | `php` | `workspace`, `release`, `baseline` |
| `ergon-lang-javascript` | `javascript` | `js` | `workspace`, `release`, `baseline` |
| `ergon-lang-typescript` | `typescript` | `js`, from `ergon-lang-javascript` | `baseline` |
| `ergon-lang-go` | `golang` | `go` | `workspace`, `release`, `baseline` |
| `ergon-lang-python` | `python` | `python` | `workspace`, `release`, `baseline` |
| `ergon-lang-rust` | `rust` | `rust` | `workspace`, `release`, `baseline` |
| `ergon-lang-terraform` | `terraform` | `terraform` | `workspace`, `release`, `baseline` |
| `ergon-lang-bash` | `bash` | `bash` | `workspace`, `release`, `baseline` |

The root package of a language module declares the name of the language, the name of the toolchain it declares, and `Register`, which registers the toolchain and then the language. A language module that declares a toolchain has one `workspace` package and one package per command it supports. `ergon-lang-typescript` and `ergon-lang-kotlin` have only `baseline`, because the JavaScript and Java modules discover and release their packages. The package for `init` is `baseline`, because the compiler rejects an import of a package named `init` unless the import renames it. `lang/go/release` also runs git, through `golang.org/x/mod/zip.CreateFromVCS`.

`ergon-lang-go` requires `golang.org/x/mod`. A language module's third-party dependencies appear only in its own `go.sum`.

The root package of the Go module cannot be called `go`, because `go` is a keyword. It is `package golang`, imported as `go.dokimi.dev/ergon/lang/go`.

## The root module

| Package | Contains | Imports |
|---|---|---|
| `internal/app` | The composition root: `Register`, which registers each language module in a fixed order, and the join of `release` to `forge` | `core/language`, every language module |
| `internal/cli` | The command tree on cobra, one file per command, and the configuration on viper: `.ergon.yaml` or the file of `--config` | `core/*`, `service/*`, cobra, viper |
| `internal/buildinfo` | Build metadata stamped at link time | stdlib |
| `cmd/ergon` | A shim that forwards an exit code | `internal/app`, `internal/cli`, `internal/buildinfo` |

This is the only module that names a language. Registration is an explicit call in `internal/app`, never an `init` with a blank import. The `action/` directory beside `cmd/` contains the four composite GitHub Actions.

## File conventions inside a package

- Name a file after the unit or family it contains, never after a technical layer. No `types.go`, `interfaces.go` or `helpers.go`.
- A banner comment that groups declarations inside a file marks a file boundary. Split those groups into files.
- `doc.go` contains the package comment: a `Package <name> <verb phrase>` first line, a `# Heading` per concern, doc links to the package's own symbols, and a `# Dependency position` section naming what it imports.
- Tests are black-box: the test package is `<pkg>_test`.
- One test file per production file, paired in both directions. A file with only type declarations still gets its twin, covering the zero values and the contracts its docblocks state.
- One `Test<Unit>` function per production file. Everything below it is `t.Run`, and the path reads `Test<Unit>/<Method>/<case>`. Error cases are cases under their method, not a group of their own.
- `t.Parallel()` at every level.
- Every exported declaration has a docblock that states its contract: what it returns, what the zero value means and what it does on failure. State a convention shared across many types once, under a `doc.go` heading.
- Benchmarks and `Test<Unit>Allocs` tests cover hot and warm paths only. A function that runs once per process, such as a registration, has neither.
