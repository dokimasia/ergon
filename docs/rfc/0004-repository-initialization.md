---
rfc: 0004
title: Repository initialization
author: Roy Klopper
status: Draft
created: 2026-10-05
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0004: Repository initialization

## Summary

`ergon init` sets up a production-grade repository for any combination of eleven languages: C#, Java, Kotlin, PHP, JavaScript, TypeScript, Go, Python, Rust, Terraform and Bash. A repository gets the common files once, the GitHub files once, and each language's toolchain configuration. `init` configures `.ergon.yaml` for the chosen languages. It renders the files it manages from templates embedded in the binary, and records its answers and the digest of each managed file in `.ergon/init.lock`. `ergon init check` fails when a managed file has been edited by hand or no longer matches the baseline of the installed ergon, and `ergon init sync` brings the files to that baseline.

## Motivation

The earlier Go-only ergon wrote ten templates once, skipped files that existed, overwrote them with `--force`, and never read them again. The repositories set up that way have drifted apart. Across 18 repositories:

| File | Repositories | Distinct versions |
|---|---|---|
| `.github/workflows/ci.yml` | 16 | 14 |
| `.gitignore` | 16 | 14 |
| `.markdownlint.yml` | 15 | 7 |
| `.github/ISSUE_TEMPLATE/config.yml` | 13 | 12 |
| `.editorconfig` | 12 | 3 |
| `.golangci.yml` | 10 | 10 |
| `.pre-commit-config.yaml` | 10 | 9 |

The drift is not limited to settings that differ between repositories. It also leaves the standards themselves unevenly enforced:

- Of the 54 golangci-lint linters that at least one Go repository enables, 18 are enabled in all 10. `paralleltest` is enabled in 3 of 10, `testpackage` in 1 and `nolintlint` in 1.
- The coverage gates are 95% for lines in TypeScript and Python, and 90% for branches in TypeScript. The Rust and Java repositories have no coverage gate. The standard is 100%.

Each repository started from a correct copy. The copies stopped receiving the improvements made in other repositories, and nothing reported that they had. Eleven languages multiply the number of templates that can drift.

## Detailed design

### Commands

| Command | Writes | Exit status |
|---|---|---|
| `ergon init new` | The common files, the GitHub files, each language's files, `.ergon.yaml` and the lock | 1 when a target file exists with other content |
| `ergon init add <language>...` | The language's files, its contributions to the shared files, and its section of `.ergon.yaml` | 1 when a target file exists with other content |
| `ergon init remove <language>...` | Deletes the language's managed files, its contributions and its section of `.ergon.yaml` | 1 when one of those files was edited by hand |
| `ergon init check` | Nothing | 1 when a managed file is missing, edited or outdated |
| `ergon init sync` | Rewrites every outdated managed file that was not edited by hand | 1 when an edited file blocks a rewrite |

`ergon init` without a subcommand prints the subcommands and exits 2. `--force` on `new`, `add` and `sync` overwrites edited and existing files. Each command that writes prints every path it wrote. `check` accepts `--json`. `add` writes the files of a shared toolchain with the first language that names it, and `remove` deletes them with the last.

### Answers

`new` takes the answers as flags and records them in the lock:

| Answer | Flag | Used for |
|---|---|---|
| Name | `--name`, defaulting to the directory's name | `README.md`, `.ergon.yaml` |
| Languages | `--language`, repeated | Which producers run |
| Owner | `--owner` | `LICENSE`, the license section of `.ergon.yaml` |
| License | `--license`, an SPDX identifier | `LICENSE`, `NOTICE`, the license section of `.ergon.yaml` |
| Repository | `--repository <owner>/<name>` | The GitHub files, the changelog links |
| Security contact | `--security-contact` | `SECURITY.md`, `CODE_OF_CONDUCT.md` |

`sync` accepts the same flags. A changed answer updates the lock, re-renders every managed file that uses it, and rewrites the matching keys of `.ergon.yaml`. `add` and `remove` change the languages.

### File classes

| Class | Written by | Checked |
|---|---|---|
| Managed | `new`, `add`, `sync` | Yes, against the lock |
| Configured | `new`, `add`, `remove`, `sync` with a changed answer | No |
| Seeded | `new`, `add`, when the file is absent | No |

- **Managed:** ergon's rendering, line for line. The repository changes it only through the answers and its local file.
- **Configured:** `.ergon.yaml`, the configuration of every ergon command. `init` writes the sections of the chosen languages and the keys its answers determine. It changes nothing else, so every other key in the file belongs to the repository.
- **Seeded:** a starting text whose content describes the repository, which the repository maintains from then on.

Each managed file that has a comment syntax opens with a comment naming its producer: `Managed by ergon init. Add repository settings to .ergon/local/<path> and run ergon init sync.` A JSON file has no comment, and the lock alone records it.

### Common files

| File | Class | Content |
|---|---|---|
| `.editorconfig` | Managed | Encoding, line endings and indentation, with a section per language |
| `.gitattributes` | Managed | `text=auto eol=lf`, and diff and generated-file rules per language |
| `.gitignore` | Managed | Editor and OS files, and each language's build output and caches |
| `.markdownlint.yml` | Managed | The Markdown rules |
| `.pre-commit-config.yaml` | Managed | File hygiene hooks and the commit-message hook |
| `Makefile` | Managed | `fmt`, `lint`, `test` and `check`, each calling the targets of every language |
| `LICENSE` | Managed | The license text for the SPDX answer, with the owner and the year |
| `NOTICE` | Managed | Present for Apache-2.0 only |
| `CODE_OF_CONDUCT.md` | Managed | The Contributor Covenant, with the security contact |
| `.changeset/config.json`, `.changeset/README.md` | Seeded | The release configuration |
| `.ergon.yaml` | Configured | The settings of every ergon command for the chosen languages |
| `README.md`, `CONTRIBUTING.md`, `SECURITY.md` | Seeded | The repository's description, contribution rules and disclosure policy |
| `docs/README.md` and the index of `docs/adr`, `docs/rfc`, `docs/architecture` and `docs/roadmap` | Seeded | The documentation tree |

### GitHub files

| File | Class | Content |
|---|---|---|
| `.github/workflows/ci.yml` | Managed | The gate on every pull request, on the merge queue and on `main` |
| `.github/workflows/release.yml` | Managed | The release flow: select-mode, version, pack and publish |
| `.github/workflows/security.yml` | Managed | CodeQL, dependency review and the OpenSSF Scorecard |
| `.github/workflows/baseline.yml` | Managed | A scheduled check against the newest ergon release |
| `.github/actions/setup-ergon/action.yml` | Managed | Installs the ergon release that the lock names and verifies its checksum |
| `.github/dependabot.yml` | Managed | Weekly updates for GitHub Actions and each language's package manager |
| `.github/ISSUE_TEMPLATE/config.yml`, `bug.yml`, `feature.yml` | Managed | Issue forms, with blank issues off and security reports sent to the advisory form |
| `.github/PULL_REQUEST_TEMPLATE.md` | Managed | The summary, the changeset and the gate checklist |
| `.github/CODEOWNERS` | Seeded | The repository's reviewers |

The workflows:

| Workflow | Triggers | Jobs | Permissions |
|---|---|---|---|
| `ci.yml` | `pull_request`, `merge_group`, `push` to `main` | `check-<language>` for each language, `docs` (Markdown lint), `license` (`ergon license check`), `baseline` (`ergon init check`), and on pull requests `changeset` (`ergon release status --since`) and `commits` (commit message check) | `contents: read` |
| `release.yml` | `push` to `main` | select-mode, version, pack and publish, as specified for the release command | Per job, as specified for the release command |
| `security.yml` | `pull_request`, `push` to `main`, weekly | `codeql` for each language CodeQL analyzes, `dependency-review` on pull requests, `scorecard` weekly | `security-events: write` on `codeql` and `scorecard`, `id-token: write` on `scorecard`, `contents: read` elsewhere |
| `baseline.yml` | Weekly, `workflow_dispatch` | Installs the newest ergon release, runs `ergon init check`, and opens an issue when the baseline is outdated and no such issue is open | `contents: read`, `issues: write` |

Every workflow follows these rules:

- The top-level `permissions` is `{}`. Each job grants only the scopes its table row lists.
- Every action is pinned to a full commit SHA, with its version in a comment. Dependabot updates both.
- `actions/checkout` runs with `persist-credentials: false`.
- `concurrency` groups by workflow and ref, and cancels a superseded run on pull requests only.
- Every job has `timeout-minutes`.
- Toolchain versions come from the repository's pin files, never from the workflow.
- No workflow uses `pull_request_target`, and no pull request job receives a secret.
- Each `check-<language>` job runs `make check-<language>`, so CI and a local run execute the same commands.

### Language contributions

Each language produces its own files and contributes fragments to the shared files. A toolchain that two languages share produces the build files they share, once. The Java module renders the Gradle wrapper and the root build files of `jvm` for Java, Kotlin or both. The JavaScript module renders the root `package.json` of `js` for JavaScript, TypeScript or both.

The fragments each language contributes:

| Shared file | What a language contributes |
|---|---|
| `.gitignore` | Its build output, caches and tool directories |
| `.editorconfig` | A section for its file types |
| `.gitattributes` | Diff and generated-file rules for its file types |
| `Makefile` | `fmt-<language>`, `lint-<language>`, `test-<language>` and `check-<language>` |
| `.github/workflows/ci.yml` | The `check-<language>` job, with its toolchain setup |
| `.github/workflows/security.yml` | Its CodeQL language, where CodeQL analyzes it |
| `.github/dependabot.yml` | Its package manager |
| `.ergon.yaml` | Its section |

The renderer concatenates the fragments in a fixed order. The common fragment comes first, and the languages follow in the order of the list in the summary. A shared toolchain comes before its first language. Two renderings of the same answers and local files produce the same bytes. A producer that writes a file that another producer also writes is a defect in ergon. A test over every pair of languages rejects it before release.

`make check` is the repository's single gate.

### Each language's gate

Each language module declares the tools that fill these slots, pins their versions, and renders their configuration:

| Slot | Requirement |
|---|---|
| Formatter | Runs as a check that fails on unformatted files |
| Linter | The strictest ruleset the tool documents, with warnings as errors |
| Type checker or static analyzer | Its strictest mode, where the language has one |
| Tests | Run in CI on every pull request |
| Coverage | Fails below 100% of lines and 100% of branches |
| Vulnerability scan | Scans the locked dependencies, or the configuration for Terraform |

Coverage does not apply to Terraform, so its gate has no coverage slot. Mutation testing is not part of this gate: the dokimi addon supplies it for every language.

### Several languages in one repository

The workspace file of each language is at the repository root: `go.work`, `package.json`, `pyproject.toml`, `Cargo.toml`, `settings.gradle.kts`, `Directory.Build.props`, `composer.json`. Their names differ, so they coexist. Java and Kotlin share one Gradle build, which the Java module renders. A language's packages are in directories the repository chooses.

`ergon init` writes no source code. A language's gate runs once the repository has its first package.

### Local files

A repository adds its own settings to a managed file through `.ergon/local/<path>`, such as `.ergon/local/.golangci.yml` for depguard rules:

- Where the tool reads a second configuration file through its own `extends` or `include` setting, the managed file references the local file, and the tool combines them.
- For any other YAML, TOML or JSON file, ergon merges the local file into the rendered document. Maps are merged key by key, and lists are appended.
- For a line-based file, ergon appends the local file's lines after the rendered content.

ergon never writes a local file. A local file for a path that is not managed is an error.

### The lock

`.ergon/init.lock` is JSON and is committed:

```json
{
  "ergon": "1.4.0",
  "answers": {
    "name": "techne",
    "languages": ["go", "typescript"],
    "owner": "ThesmOS B.V.",
    "license": "MIT",
    "repository": "dokimasia/techne",
    "security-contact": "security@thesmos.sh"
  },
  "files": [
    { "path": ".editorconfig", "producer": "common", "sha256": "4be1…" },
    { "path": ".github/workflows/ci.yml", "producer": "github", "sha256": "0c3a…" },
    { "path": ".golangci.yml", "producer": "go", "local": "9e41…", "sha256": "77d0…" }
  ]
}
```

| Finding | Condition | `check` | `sync` |
|---|---|---|---|
| `missing` | A file in the lock is absent | Fails | Writes it |
| `edited` | The file's digest differs from the lock | Fails | Reports a conflict and leaves the file, unless `--force` |
| `outdated` | The current rendering differs from the lock, and the file is unedited | Fails | Rewrites it and updates the lock |

`local` is the digest of the file's local file. A changed local file makes its managed file `outdated`.

### Tool versions

Each ergon release embeds the tool versions its baseline was tested with: actions, linters, formatters, test and coverage tools. Upgrading ergon and running `ergon init sync` moves a repository to them. Dependabot updates dependency manifests and lockfiles, and the action pins in the workflows. A pin it changes in a managed file makes that file `edited`, so `ci.yml`'s `baseline` job fails until the next ergon release contains the same pin.

`baseline.yml` reports a repository whose baseline is behind the newest ergon release. A repository may lag behind the baseline. The issue records that it does.

### Packages

| Package | Contains | Imports |
|---|---|---|
| `core/language/init.go` | The `Initializer` role, `File`, `Class` and `Answers` | position 0 |
| `ergon-service/baseline` | Composition, rendering, the merge of local files, the lock, and `New`, `Add`, `Remove`, `Check` and `Sync` | `core/*`, `baseline/common`, `baseline/github` |
| `ergon-service/baseline/common` | The common templates | stdlib |
| `ergon-service/baseline/github` | The GitHub templates, and the CI rules that every workflow follows | stdlib |
| `ergon-lang-<language>/baseline` | The language's templates, its gate tools and their pinned versions, and its fragments | `core/*`, `ergon-lang` |

The packages for `init` are named `baseline`, because the compiler rejects an import of a package named `init` unless the import renames it. The `baseline` packages of the Java and JavaScript modules also render the build files of the toolchain that Kotlin and TypeScript share with them.

```go
// Initializer renders the files that a producer contributes to a
// repository. Every language implements it. A toolchain that two languages
// share implements it for the build files those languages share.
type Initializer interface {
	// Files returns the files and fragments the producer contributes for
	// a. It reads no file and runs no command, so a rendering depends on
	// a and on the ergon release alone.
	Files(a Answers) ([]File, error)
}

// File is one file a producer renders.
type File struct {
	// Path is repository-relative and slash-separated.
	Path string

	// Class is Managed, Configured or Seeded.
	Class Class

	// Content is the whole rendering of a file the producer writes alone.
	// It is nil when Fragment is set.
	Content []byte

	// Fragment is the producer's part of a shared file. It is nil when
	// Content is set.
	Fragment []byte
}
```

### Failure handling

| Failure | State afterwards | Recovery |
|---|---|---|
| `new` finds an existing file with other content | Nothing written | Remove the file, or pass `--force` and review the diff |
| `new` finds `.ergon/init.lock` | Nothing written | Use `add` or `sync` |
| An unknown language | Nothing written | The error lists the eleven languages |
| A local file for a path that is not managed | Nothing written | The error names the local file |
| `sync` meets an edited file | Every other outdated file is rewritten. The edited file is unchanged | Move the edit into the local file, then run `sync --force` |
| `remove` meets an edited file | Nothing removed | The same |

Every command computes its whole change before it writes. It writes each file to a temporary name in the same directory and renames it, so an interrupted run leaves each file either old or new.

### Adopting an existing repository

`ergon init new --force` in a repository that predates ergon's baseline overwrites every managed file. The diff then shows each repository-specific setting the baseline lacks. Each such setting moves into a local file before the commit.

## Alternatives considered

### A. Generate once

The earlier ergon wrote the files once and left them to the repository.

**Why not:** the 18 repositories it set up have up to 14 distinct versions of one file. Standards such as `paralleltest` and 100% coverage are enforced in only some of them. One-shot copies stop receiving improvements, and nothing reports it.

### B. Three-way merge updates

copier records the template version and the answers. `copier update` merges the newer template into the edited files.

**Why not:** a merge keeps every hand edit, including edits that weaken a gate, and nothing reports the difference from the standard. A conflict leaves markers in a configuration file that a person resolves by hand, in every repository at every update.

### C. Answers in `.ergon.yaml`

`init` would read its languages, owner and extensions from a section of `.ergon.yaml`.

**Why not:** `init` writes `.ergon.yaml` for the chosen languages. A file that is both `init`'s output and its input has no single owner for any key. The answers belong to the lock, and the repository's additions to the local files.

### D. Shared configuration through each tool's `extends` alone

Each repository would reference a published base configuration instead of a copy.

**Why not:** several files in the baseline have no such mechanism, among them `.editorconfig`, `.gitignore`, the Makefile and the workflows, so they would still be copies. ergon uses a tool's `extends` where it exists, for the local file, and manages the base file itself.

### E. A template repository per language

A GitHub template repository per language, copied at creation.

**Why not:** it has no update path at all, and it cannot combine several languages in one repository.

## Drawbacks

- A hand edit to a managed file fails `check`. Every repository-specific setting has to be expressed in a local file.
- A repository receives a baseline change only through an ergon upgrade and a `sync` commit.
- A Dependabot pull request that bumps an action fails `ci.yml`'s `baseline` job until an ergon release contains the same pin.
- ergon maintains templates and pinned tool versions for eleven languages. Each language needs a fixture repository that ergon's CI generates and then gates with that language's own toolchain, nine toolchains in all.
- The merge of a local file has fixed semantics: maps merge, lists append. A setting that needs a list element removed has no expression and needs a change to the baseline.
- The Makefile requires `make` on every machine that runs the gate.

## Open questions

- Which of the eleven languages does CodeQL analyze? This design assumes C#, Go, Java, Kotlin, Python, Rust, JavaScript and TypeScript, from memory.
- Does Dependabot update every package manager of the eleven languages? This design assumes it does, from memory.

## Unresolved and future work

- Repository settings on GitHub, such as branch protection, rulesets, environments and required checks, are not proposed.
- Mutation testing gates are not proposed. The dokimi addon supplies them.

## References

| What | Where |
|---|---|
| The earlier ergon's `init` and its templates | `go.thesmos.sh/ergon`, `internal/scaffold/scaffold.go` and `internal/scaffold/templates` |
| copier's update and three-way merge | https://copier.readthedocs.io/en/stable/updating/ |
| The drift measurement | `~/.cache/ergon-init/common.py` and `~/.cache/ergon-init/linters.py`, run on 2026-10-05 |
