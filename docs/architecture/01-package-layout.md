<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Package layout

Where a file goes, and what each package may import. The module boundaries and the rules between modules are argued in RFC-0001. This document covers the packages inside them.

## Two rules place every file

**Each command is split in two.** The part that is the same for every language is in `ergon-service`. The part that differs is in a language module, or in `ergon-lang` when two languages share it. `core` declares the roles that connect the two parts.

**Machinery is placed by reuse.** A package belongs in `ergon-lang` when more than one language uses it, and in a language module when only one does. Rust and Python both read TOML manifests, so TOML editing is in `lang/manifest`. `go.mod` handling serves Go alone, so it is in `ergon-lang-go`.

A producer of `init` configures its own concern alone. The options of a language, their rules, its templates, its jobs in CI and the programs that its gate runs are in its module. The service renders, resolves and runs them for every producer alike.

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
| `spdx` | `ID`, the identifiers of the 44 licenses of RFC-0003, and `ID.Valid` | stdlib |
| `workflow` | `Job`, `Setup`, `Step`, `Action`, `Assets`, `CodeQL`, `Update` and `Contribution`: a producer's part of the GitHub workflows | stdlib |

A type belongs here when a role signature names it. A type that does something, or that only one command reads, belongs in that command's package.

### Position 1: options, toolchains, languages and roles

| Package | Contains | Imports |
|---|---|---|
| `option` | The option types that every section of `.ergon.yaml` composes, each with `Validate`: the kinds of tool `Module`, `PyPI`, `NPM`, `Crate`, `Maven`, `Composer` and the release binaries `Binary`, `Release` and `UV`, the module plugins of golangci-lint in `Plugins` and in a group that implements `Linters`, the step options `Run`, `Generate`, `Fuzz`, `Bench`, `Mutate`, `Audit` and `Threshold`, `Check`, `Step` with the target that the gate runs for it, `Severity`, `Paths`, `Version`, and `CI` with its forms for the runners and the versions. It also declares the keys of the struct tags of an options struct, such as `source`, the registry of a `Version` that a baseline update resolves | `workflow` |
| `language` | `Toolchain`, `Declaration`, `Catalog`, `RegisterToolchain`, `Register`, `ToolchainRole` and `Role`, and the role interfaces with one file per command. `release.go` contains `Resolution`, `Versioner`, `Tagger`, `Packer` with the directories `AssetsDir` and `CasksDir` of a pack, `Publisher`, `Locker` and `Edit`. `init.go` contains `Producer`, `Calculator`, `Configurable`, `Contributor`, `Placer`, `LocalChecker`, `File`, `Options`, `Answers` with `Validate`, `Repository`, and the path of `.ergon.yaml` | `workspace`, `spdx`, `workflow` |

The catalog has two kinds of entry, and each registers its roles. Toolchains discover packages and implement the release roles. Each language declares its toolchain and implements the roles of `init`. A toolchain that two languages share also implements them, for the files and the contributions those languages share: the `jvm` toolchain of the Java module and the `js` toolchain of the JavaScript module. Roles are separate interfaces, so a toolchain or a language declines a command by not implementing its role. The catalog selects by type assertion, and each implementation asserts its roles at compile time.

## ergon-service

The language-neutral side of each command. It imports `core` and nothing else in the repository.

| Package | Contains | Imports |
|---|---|---|
| `release` | The changesets of a repository, the planner, the status of a branch, the version of a plan with its changelogs, the version pull request and the mark of its commit, the publish plan, the stale lockfiles of a publish plan and their rewrite, the pack and the publish with the assets of each release, the casks of a tap, the verification that the gate passed on the content of a commit, the decision that a passed run covers the content of a run of the gate, and the interfaces `Forge`, `Proposer`, `Releaser`, `TagForge`, `TapForge`, `GateForge`, `SkipForge` and `Marker` that it calls | `core/*`, `forge` for the type of a release, `vcs`, doublestar |
| `licenses` | `Config` and `Directory`, the `license` section of `.ergon.yaml`, `Text` and the texts of the 44 licenses, `Check`, `Fix`, the table of comment styles with its overrides, and the removal of outdated header blocks | `core/*`, `vcs`, skywalking-eyes `assets`, `pkg/comments`, `pkg/header`, `pkg/license` and `pkg/logger`, logrus, doublestar |
| `licenses/baseline` | The producer of `LICENSE` and `NOTICE` for `init`, at the root and in each directory of `license.directories` | `core/*`, `licenses` |
| `baseline` | The `init` command: `New`, `Add`, `Remove`, `Check`, `Sync`, `Options`, `Sections` and `Overrides` over the producers that `Open` receives, the plan of their changes, the local check of each producer, and the refusal of a lock that a newer release of ergon wrote | `core/*`, `baseline/lock`, `baseline/options`, `baseline/overlay`, `baseline/render`, `pin` |
| `baseline/lock` | The format of `.ergon/init.lock` | `core/*` |
| `baseline/options` | The sections of `.ergon.yaml`: `Fields`, the keys of the options of a producer, the resolution against the record and the answers of the lock, the strict decode into the struct of a producer, its `Validate`, and the sections with the comments of the `doc` tags | `core/*`, viper and mapstructure for the decode, go.yaml.in/yaml/v3 for the writer |
| `baseline/overlay` | The local files: the merge of YAML and the appended text | `core/*`, go.yaml.in/yaml/v3 |
| `baseline/render` | The engine of the templates, the classes from the template tree, the files of a `Placer`, the collection of the contributions, and the join of fragments | `core/*` |
| `baseline/common` | The producer of the common files and the section `common` | `core/*`, `release` for the branch of the version pull request |
| `baseline/github` | The producer of the GitHub files and the section `github`, which renders the contributions of every producer with the cache and the prune of the tools of each job, the job that prunes the tool caches, and its check of the local file of `dependabot.yml` | `core/*`, go.yaml.in/yaml/v3 |
| `baseline/baselinetest` | The test kit of the producers: `New` renders them into a directory of a test as `init new` does, and `Hygiene` checks the format of each file | `core/*`, `baseline`, assert, go.yaml.in/yaml/v3, go-toml |
| `pin` | The pins of the options of a producer, which `Find` lists by the types of their fields, and `Resolver`, which resolves the newest release of each pin in the module proxy, PyPI, npm, crates.io, Maven Central, Packagist, Chocolatey or the releases of GitHub | `core/*`, `baseline/options`, `forge`, `golang.org/x/mod` |
| `tool` | `ergon tool run`: the installation of a tool into the cache, the check of a release binary against its digest, the build of golangci-lint with the module plugins of its section, and the run. `RunRelease` runs a release binary that no section names, such as a release of ergon. `Runner.Prune` removes each file of the cache that is not the install of a tool of the options, for `ergon tool prune`. `PruneCaches` deletes the tool caches of GitHub Actions that newer caches of the same job replaced, for `ergon tool ci prune` | `core/language`, `core/option`, `forge` for the type of a cache, xz for the `.tar.xz` of a release binary |
| `vcs` | git: the files of a working tree, the diff since a ref, HEAD with its tree and the commit that added a file, annotated and lightweight tags, the snapshot commit, the atomic push | stdlib |
| `vcs/vcstest` | git in a test: an environment without the configuration of the user, and a working tree of a test | stdlib, assert |
| `forge` | The GitHub client: signed commits through `createCommitOnBranch` and the commit of one file, branch and tag refs, pull requests, the draft releases, the upload of their assets and their publish, the list of releases with the digests of their assets, the passed runs of a workflow, the tree and the parent of a commit, the statuses of a commit, the heads of the pull requests of a commit, the links of a commit and of a pull request, and the list and the deletion of the caches of GitHub Actions | stdlib |
| `workspace` | Which toolchains and languages are active in a repository | `core/language`, `core/workspace`, `vcs` |

`vcs` is the only package in this module that runs git, and `forge` is the only package that calls the GitHub API. `forge` satisfies the interfaces of `release`, `pin` and `tool` without importing any of them. `release` and `pin` import `forge` for the type of a release of GitHub, and `tool` imports it for the type of a cache. `licenses` is the only package in the repository that imports skywalking-eyes. The `baseline` packages name no language and no toolchain.

## ergon-lang

Machinery that two or more languages use. It imports `core` and nothing else in the repository.

| Package | Contains | Imports |
|---|---|---|
| `manifest` | Byte-range edits for JSON, TOML, XML and properties files that keep formatting and comments | stdlib, go-toml v2 |
| `command` | The subprocess runner | stdlib |
| `conformance` | One suite per role. Every toolchain and language runs it against its fixtures | `core/language`, `core/workspace` |

A language module imports `manifest`, `command` and `conformance`. The only imports between language modules are two: `ergon-lang-typescript` imports the root package of `ergon-lang-javascript`, and `ergon-lang-kotlin` imports the root package of `ergon-lang-java`, for the name of their toolchain. The tests of a language module also import `service/baseline`, whose test kit renders the language with its options at the baseline, as every command renders them. The tests of the release roles of a language module import `service/vcs` and its test kit, which run git in the working trees of the tests.

## A language module

```text
ergon-lang-go/
  doc.go        package golang
  language.go   Language, Toolchain, Git and Register
  workspace/    go.work and go.mod discovery, for every command
  release/      Versioner: require rewrites, go.sum through the file proxy, the
                replaces of go.work, tag prefixes. Locker: the go.sum lines of a
                pending release that record other content, and their rewrite.
                Packer: the binaries of the commands of a module through GoReleaser
  baseline/     the producer: the options of the section go and their rules, the
                templates, the configuration of GoReleaser of each module with a
                command, and the contributions to the workflows
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

The root package of a language module declares the name of the language, the name of the toolchain it declares, and `Register`, which registers the toolchain and then the language. The Java and JavaScript modules register their toolchain with the producer of the files and the contributions that Kotlin and TypeScript share with them. A language module that declares a toolchain has one `workspace` package and one package per command it supports. `ergon-lang-typescript` and `ergon-lang-kotlin` have only `baseline`, because the JavaScript and Java modules discover and release their packages. The package for `init` is `baseline`, because the compiler rejects an import of a package named `init` unless the import renames it.

`lang/go/release` also runs git, through `golang.org/x/mod/zip.CreateFromVCS`. The Go module reads the tags, snapshots the working tree, reads HEAD and creates the tag of a module with commands through the functions of `Git`. The root module fills `Git` with `vcs.Tags`, `vcs.Snapshot`, `vcs.Head` and `vcs.LightTag`. The Go module runs GoReleaser through `Tools`, which the root module fills with `ergon tool run` of the running ergon, so the module does not import `service` outside its tests.

A `baseline` package contains:

- `baseline.go`: the producer, with its name, its templates, the baseline value of each option in `Options`, and its contribution for options of another type. Each pin of `Options` is a literal, in the method or in a package-level variable, so a baseline update rewrites it.
- `options.go`: the struct of the language's options, composed from `core/option`, with a `doc` tag on each field, `Validate` with the language's own rules, and `Contribution`: the job `check-<language>` with its setup, the steps that set up its toolchain in a release, its CodeQL language and its Dependabot ecosystem, as values of `core/workflow`.
- `toolchain.go` and `toolchainoptions.go` in the Java and JavaScript modules: the producer and the options of the toolchain that two languages share, whose contribution is the setup that the jobs of both languages run.
- `templates/managed/`, `templates/seeded/` and `templates/shared/`: the templates, whose directory states their class. The Java and JavaScript modules have a tree for the language and a tree for the toolchain, such as `templates/java/` and `templates/jvm/`.

`ergon-lang-go` requires `golang.org/x/mod` and go.yaml.in/yaml/v3. A language module's third-party dependencies appear only in its own `go.sum`.

The root package of the Go module cannot be called `go`, because `go` is a keyword. It is `package golang`, imported as `go.dokimi.dev/ergon/lang/go`.

## The root module

| Package | Contains | Imports |
|---|---|---|
| `internal/app` | The composition root: `Register`, which registers each language module in a fixed order, with the functions of `service/vcs` that the Go module reads and tags a repository through, and the runner of the tools of the section go through `ergon tool run` | `core/language`, `service/vcs`, every language module |
| `internal/cli` | The command tree on cobra, one file per command, and the configuration on viper: `.ergon.yaml` or the file of `--config`. `init.go` runs `ergon init`, `upgrade.go` runs `ergon init upgrade` and `ergon init ci upgrade`, `license.go` runs `ergon license`, `release.go` runs `ergon release` and joins `release` to the GitHub client of `forge`, and `tool.go` runs `ergon tool run`, `ergon tool prune` and `ergon tool ci prune`, which joins `tool` to the same client. `repository.go` returns the producers of the common files, the GitHub files and the license files with `Producers`, opens a repository with them before the languages, and finds the repository of the working directory. These commands read no configuration through viper: `baseline/options` reads the options of `.ergon.yaml` | `core/*`, `service/*`, cobra, viper |
| `internal/rewrite` | `Apply`, which writes resolved releases into the literals of the `Options` method of a producer | `core/option`, `core/workflow`, `service/pin` |
| `internal/cmd/update-baseline` | The command of ergon's repository that moves the pins of every baseline to their newest releases and opens the pull request of the change. ergon releases no binary of it | `core/*`, `internal/app`, `internal/cli`, `internal/rewrite`, `service/*` |
| `internal/buildinfo` | The version of the running binary, which the go command writes into it from the tag of the module: the version of the release, or `dev` for a build that no release tags, with the commit and its time | stdlib, `golang.org/x/mod` |
| `cmd/ergon` | A shim that forwards an exit code | `internal/app`, `internal/cli`, `internal/buildinfo` |

This is the only module that names a language. Registration is an explicit call in `internal/app`, never an `init` with a blank import. ergon does not publish GitHub Actions: the workflows of a repository install ergon with its action `setup-ergon`, as RFC-0002 states.

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
