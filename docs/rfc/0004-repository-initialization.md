---
rfc: 0004
title: Repository initialization
author: Roy Klopper
status: Accepted
created: 2026-10-05
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
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

The drift is not limited to settings that differ between repositories. It also leaves the standards themselves unevenly enforced. Of the 54 golangci-lint linters that at least one Go repository enables, 18 are enabled in all 10. `paralleltest` is enabled in 3 of 10, `testpackage` in 1 and `nolintlint` in 1.

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

`ergon init` without a subcommand prints the subcommands and exits 2. A command line without a required flag also exits 2. `--force` on `new`, `add` and `sync` overwrites edited and existing files. Each command that writes prints a line for every path it wrote or removed. `check` prints a line for every finding, or a JSON array with `--json`. `add` writes the files of a shared toolchain with the first language that names it, and `remove` deletes them with the last.

### Answers

`new` takes the answers as flags and records them in the lock:

| Answer | Flag | Used for |
|---|---|---|
| Name | `--name`, defaulting to the directory's name | `README.md`, `.ergon.yaml` |
| Languages | `--language`, repeated | Which producers run |
| Owner | `--owner`, required | `LICENSE`, the license section of `.ergon.yaml` |
| License | `--license`, required: `MIT` or `Apache-2.0` | `LICENSE`, `NOTICE`, the license section of `.ergon.yaml` |
| Year | `--year`, defaulting to the current year | `LICENSE`, `NOTICE` |
| Repository | `--repository <owner>/<name>`, required | The GitHub files, the changelog links |
| Security contact | `--security-contact`, required | `SECURITY.md`, `CODE_OF_CONDUCT.md` |

`sync` accepts the same flags but `--language`, and records the answer of each flag in the lock. It renders again every managed file that uses a changed answer, and rewrites the matching keys of `.ergon.yaml`. An answer without a flag keeps its value. `add` and `remove` change the languages.

### File classes

| Class | Written by | Checked |
|---|---|---|
| Managed | `new`, `add`, `sync` | Yes, against the lock |
| Configured | `new`, `add`, `remove`, `sync` with a changed answer | No |
| Seeded | `new`, `add`, when the file is absent | No |

- **Managed:** ergon's rendering, line for line. The repository changes it only through the answers and its local file.
- **Configured:** `.ergon.yaml`, the configuration of every ergon command. `init` writes the sections of the chosen languages and the keys its answers determine. It changes nothing else, so every other key in the file belongs to the repository. `init` takes no setting from `.ergon.yaml`. `check` does not read the file, and the commands that write parse it only to write their keys.
- **Seeded:** a starting text whose content describes the repository, which the repository maintains from then on.

Each managed file that has a comment syntax opens with a comment naming its producer: `Managed by ergon init. Add repository settings to .ergon/local/<path> and run ergon init sync.` A JSON file has no comment, and the lock alone records it.

### Common files

| File | Class | Content |
|---|---|---|
| `.editorconfig` | Managed | Encoding, line endings and indentation, with a section per language |
| `.gitattributes` | Managed | `text=auto eol=lf`, and diff and generated-file rules per language |
| `.gitignore` | Managed | Editor and OS files, and each language's build output and caches |
| `.markdownlint.yml` | Managed | The Markdown rules |
| `.commitlint.yaml` | Managed | The commit-message rules: Conventional Commits types, a subject of at most 72 characters without a final period, and body lines of at most 100 |
| `.pre-commit-config.yaml` | Managed | File hygiene hooks, `make check`, and the commit-message hook, which runs commitlint |
| `Makefile` | Managed | `fmt`, `lint`, `test`, `audit` and `check`, each calling the targets of every language |
| `LICENSE` | Managed | The license text for the SPDX answer, with the owner and the year |
| `NOTICE` | Managed | Present for Apache-2.0 only |
| `CODE_OF_CONDUCT.md` | Managed | The Contributor Covenant, with the security contact |
| `.changeset/config.json`, `.changeset/README.md` | Seeded | The release configuration |
| `.ergon.yaml` | Configured | The settings of every ergon command for the chosen languages |
| `README.md`, `CONTRIBUTING.md`, `SECURITY.md` | Seeded | The repository's description, contribution rules and disclosure policy |
| `docs/README.md` and the index of `docs/adr`, `docs/rfc`, `docs/architecture` and `docs/roadmap` | Seeded | The documentation tree |

A repository adds its closed set of commit scopes in its local file of `.commitlint.yaml`, as the rule `scope-enum`.

### GitHub files

| File | Class | Content |
|---|---|---|
| `.github/workflows/ci.yml` | Managed | The gate on every pull request, on the merge queue and on `main` |
| `.github/workflows/release.yml` | Managed | The release flow: select-mode, version, pack and publish |
| `.github/workflows/security.yml` | Managed | CodeQL, dependency review and the OpenSSF Scorecard |
| `.github/workflows/codeql.yml` | Managed | The CodeQL analysis of one language, which `security.yml` calls for each language |
| `.github/workflows/baseline.yml` | Managed | A scheduled check against the newest ergon release |
| `.github/actions/setup-ergon/action.yml` | Managed | Installs the ergon release that the lock names on a Linux, macOS or Windows runner, and checks its archive against the release's `checksums.txt` |
| `.github/actions/setup-make/action.yml` | Managed | Installs GNU make from Chocolatey on a Windows runner, whose image has no make |
| `.github/dependabot.yml` | Managed | Weekly updates for GitHub Actions and each language's package manager |
| `.github/ISSUE_TEMPLATE/config.yml`, `bug.yml`, `feature.yml` | Managed | Issue forms, with blank issues off and security reports sent to the advisory form |
| `.github/PULL_REQUEST_TEMPLATE.md` | Managed | The summary, the changeset and the gate checklist |
| `.github/CODEOWNERS` | Seeded | The repository's reviewers |

The workflows:

| Workflow | Triggers | Jobs | Permissions |
|---|---|---|---|
| `ci.yml` | `pull_request`, `merge_group`, `push` to `main` | `check-<language>` for each language, `docs` (markdownlint-cli2), `license` (`ergon license check`), `baseline` (`ergon init check`), and on pull requests `changeset` (`ergon release status --since`) and `commits` (commitlint on each commit) | `contents: read` |
| `release.yml` | `push` to `main` | select-mode, version, pack and publish, as specified for the release command | Per job, as specified for the release command |
| `security.yml` | `pull_request`, `push` to `main`, weekly | `codeql-<language>` for each language CodeQL analyzes, `dependency-review` on pull requests, `scorecard` weekly | `security-events: write` on the CodeQL jobs and `scorecard`, `id-token: write` on `scorecard`, `contents: read` elsewhere |
| `baseline.yml` | Weekly, `workflow_dispatch` | Installs the newest ergon release, runs `ergon init check`, and opens an issue when the baseline is outdated and no such issue is open | `contents: read`, `issues: write` |

`setup-ergon` downloads `ergon_<version>_<os>_<arch>.tar.gz` and `checksums.txt` from the release `v<version>` of github.com/dokimasia/ergon, where `<os>` is `linux`, `darwin` or `windows` and `<arch>` is `amd64` or `arm64`. Every release of ergon publishes these assets.

Every workflow follows these rules:

- The top-level `permissions` is `{}`. Each job grants only the scopes its table row lists.
- Every action is pinned to a full commit SHA, with its version in a comment. Dependabot updates both.
- `actions/checkout` runs with `persist-credentials: false`.
- `concurrency` groups by workflow and ref, and cancels a superseded run on pull requests only.
- Every job has `timeout-minutes`. A job that calls `codeql.yml` takes the timeout of the called job.
- Toolchain versions come from the repository's pin files, never from the workflow.
- No workflow uses `pull_request_target`, and no pull request job receives a secret.
- Every job specifies its runner image with a version, never a `-latest` label.
- Each `check-<language>` job and the `baseline` job of `ci.yml` run on a matrix of `ubuntu-26.04`, `macos-26` and `windows-2025` without `fail-fast`, so each gate and the check of the baseline run on Linux, macOS and Windows. Every other job runs on `ubuntu-26.04`.
- Each `check-<language>` job runs `make check-<language>`, so CI and a local run execute the same commands. The step runs in `bash`, which is Git Bash on Windows, after `setup-make`.

A language's check job and its CodeQL analysis run once the repository has the file that pins the language's toolchain:

| Language | Pin file | Setup |
|---|---|---|
| C# | `global.json` | `actions/setup-dotnet` |
| Java, Kotlin | `.java-version` | `actions/setup-java`, Temurin, and `actions/setup-go`, which builds osv-scanner |
| PHP | `.php-version` | `shivammathur/setup-php` |
| JavaScript, TypeScript | `package.json` | `actions/setup-node` |
| Go | `go.work` | `actions/setup-go` |
| Python | `pyproject.toml` | `astral-sh/setup-uv` |
| Rust | `rust-toolchain.toml` | `actions-rust-lang/setup-rust-toolchain` |
| Terraform | `.terraform-version` | `hashicorp/setup-terraform`, and `astral-sh/setup-uv`, which runs checkov |
| Bash | none | `astral-sh/setup-uv`, which runs the pinned shellcheck-py through `uvx` |

### Language contributions

Each language produces its own files and contributes fragments to the shared files. A toolchain that two languages share renders the fragments they share, once, before its first language. The `jvm` toolchain of the Java module renders the target `audit-jvm`, the CodeQL analysis of `java-kotlin` and the Gradle updates of Dependabot. The `js` toolchain of the JavaScript module renders `biome.json`, the targets `fmt-js`, `lint-js` and `audit-js`, the analysis of `javascript-typescript` and the npm updates. The catalog records this rendering as a role of the toolchain, as it records the roles of a language.

`ergon init` renders no build file of a toolchain, such as the Gradle wrapper or a root `package.json`. The repository writes them with its first package.

The fragments each language contributes:

| Shared file | What a language contributes |
|---|---|
| `.gitignore` | Its build output, caches and tool directories |
| `.editorconfig` | A section for its file types |
| `.gitattributes` | Diff and generated-file rules for its file types |
| `Makefile` | `fmt-<language>`, `lint-<language>`, `test-<language>`, `audit-<language>` and `check-<language>`, or `audit-jvm` and `audit-js` for a shared toolchain |
| `.github/workflows/ci.yml` | The `check-<language>` job, with its toolchain setup |
| `.github/workflows/security.yml` | Its `codeql-<language>` job, which calls `codeql.yml`, where CodeQL analyzes it: C#, Go, Python and Rust, and the shared `java-kotlin` and `javascript-typescript` |
| `.github/dependabot.yml` | Its package manager: `nuget`, `gradle`, `composer`, `npm`, `gomod`, `uv`, `cargo` or `terraform` |
| `.ergon.yaml` | Its section |

The renderer concatenates the fragments in a fixed order. The common fragment comes first, the GitHub fragment second, and the languages follow in the order of the list in the summary. A shared toolchain comes before its first language. Two renderings of the same answers and local files produce the same bytes. A producer that writes a file that another producer also writes is a defect in ergon. A test over every pair of languages rejects it before release.

`make check` is the repository's single gate.

### Each language's gate

Each language module renders one managed configuration of its linter. Its fragment of the Makefile fills these slots with commands, and pins the version of each tool that the toolchain does not include:

| Slot | Requirement |
|---|---|
| Formatter | Runs as a check that fails on unformatted files |
| Linter | The strictest ruleset the tool documents, with warnings as errors |
| Type checker or static analyzer | Its strictest mode, where the language has one |
| Tests | Run in CI on every pull request |
| Vulnerability scan | Scans the locked dependencies, or the configuration for Terraform |

Coverage and mutation testing are not part of this gate. The dokimi addon supplies mutation testing for every language.

The linter configuration of each language:

| Language | Managed file | Linter | Rules |
|---|---|---|---|
| C# | `.globalconfig` | The .NET analyzers, through `dotnet build` and `dotnet format` | Every rule of the latest analysis level as an error, with nullable references and XML documentation |
| Java | `gradle/ergon-java.init.gradle.kts` | PMD 7.28.0, javac and javadoc, through Gradle's `--init-script` | PMD's base ruleset, and every warning of javac and doclint as an error |
| Kotlin | Its section of `.editorconfig` | ktlint 1.8.0 | The official code style of Kotlin, with ktlint's experimental rules |
| PHP | `phpstan.dist.neon` | PHPStan 2.2.17 with phpstan-strict-rules 2.0.12, and PHP-CS-Fixer 3.95.27 | Level max with every check that the level leaves off, and the PER coding style |
| JavaScript, TypeScript | `biome.json` | Biome 2.5.15, and tsc 7.0.2 for TypeScript | Every stable group of rules as an error, with the project and types domains, and the type-safety options of tsc |
| Go | `.golangci.yml` | golangci-lint v2.14.0 | The 54 linters that a Go repository of the baseline enables, each with the strictest setting that one of them uses |
| Python | `ruff.toml` | ruff 0.16.10, and mypy 2.4.0 | Every stable rule of ruff, and mypy's strict mode with twelve more error codes |
| Rust | `clippy.toml` | clippy of the repository's toolchain | Every pedantic lint, and the lints of documentation, as errors |
| Terraform | `.tflint.hcl` | tflint 0.64.0 | Every rule of the terraform ruleset that tflint bundles |
| Bash | `.shellcheckrc` | shellcheck 0.11.0, through shellcheck-py 0.11.0.1 | Eight optional checks |

A rule set is complete unless the tool's own documentation advises against a part of it:

- PMD's guide states that every rule of every category reports a huge number of violations, most of them unimportant. Its base ruleset reported 32 findings in a Java repository of the baseline, against 1,910 for every category.
- clippy's documentation advises against enabling the restriction group whole, and calls the nursery group unfinished.
- Biome's nursery group, ruff's preview rules and PHPStan's bleeding edge change between patch releases.

The configurations contain these exceptions:

- **Go:** depguard allows only the standard library in a repository without rules, so the managed rule denies the packages that the standard library replaces and allows the rest. A repository adds the depguard rules of its module boundaries and the gci section of its own modules in its local file.
- **Python:** S101, PLR2004 and INP001 skip the tests of pytest, which assert with `assert`, state literal values and are in a directory without `__init__.py`. CPY001 accepts the copyright notice of ergon's license header, which puts the owner before the year.
- **C#:** the analysis keeps file-scoped namespaces, which the templates of the .NET SDK declare.
- **JavaScript and TypeScript:** `useLiteralKeys` is off, because it contradicts tsc's `--noPropertyAccessFromIndexSignature`. `noNodejsModules` is skipped, because it is for code that runs in a browser.
- **Rust:** clippy does not read a lint level from `clippy.toml`, so `lint-rust` sets the levels on the command line.
- **JSON:** `biome.json` is JSON, so it opens with no managed comment, and a local file cannot extend it.

The vulnerability scan of each language is a target of the Makefile that `check-<language>` requires:

| Language | Target | Scan | Fails on |
|---|---|---|---|
| C# | `audit-csharp` | `dotnet restore --force`, with NuGet's audit of every package and its warnings NU1900 to NU1904 as errors | A known vulnerability of any severity, or a vulnerability source that cannot be reached |
| Java, Kotlin | `audit-jvm` | osv-scanner v2.6.0 through `go run`, over every `gradle.lockfile` and `buildscript-gradle.lockfile` | A known vulnerability, or a repository without a Gradle lockfile |
| PHP | `audit-php` | `composer audit --locked` | A known vulnerability, or an abandoned package |
| JavaScript, TypeScript | `audit-js` | `npm audit --audit-level=low --include=dev` | A known vulnerability of any severity |
| Go | `audit-go` | govulncheck v1.8.0 through `go run`, once per module of `go.work` | A known vulnerability that the code calls |
| Python | `audit-python` | pip-audit 2.10.1 through `uvx`, over the packages of `uv.lock` that `uv export` writes to a `pylock.toml` | A known vulnerability, or a package that it cannot check |
| Rust | `audit-rust` | cargo-audit 0.22.2, which `cargo install --locked` installs | A known vulnerability |
| Terraform | `audit-terraform` | checkov 3.3.23 through `uvx`, on Python 3.13 and with the packages published before 2026-10-06 | A failed check, or a file that does not parse |
| Bash | none | Bash has no locked dependencies | |

The jvm toolchain renders `audit-jvm`, and the js toolchain renders `audit-js`, once for both of their languages. A Java or Kotlin repository locks the dependencies of its Gradle build, because Gradle writes a lockfile only for a build with dependency locking, and ergon init renders no build file that enables it.

pip-audit scans the export of `uv.lock`, because `uv audit` is a preview command whose interface can change in any uv release. checkov scans the configuration of Terraform, because Trivy, the other scanner that covers it, takes 89 s to build with `go run` and leaves a 2.6 GB build cache, against 4 s for checkov through `uvx`.

### Several languages in one repository

The workspace file of each language is at the repository root: `go.work`, `package.json`, `pyproject.toml`, `Cargo.toml`, `settings.gradle.kts`, `Directory.Build.props`, `composer.json`. Their names differ, so they coexist. Java and Kotlin share the Gradle build of `settings.gradle.kts`. A language's packages are in directories the repository chooses. The targets of Go lint and test every module that `go list -m` lists, which is every module of `go.work`.

`ergon init` writes no source code. A language's gate runs once the repository has its first package.

### Local files

A repository adds its own settings to a managed file through `.ergon/local/<path>`, such as `.ergon/local/.commitlint.yaml` for its commit scopes:

- For a YAML file, ergon merges the local file into the rendered document. Maps are merged key by key, with the local value for a key that both have, and lists are appended.
- For any other file, ergon appends the local file's lines after the rendered content.

ergon never writes a local file. A local file for a path that is not managed is an error.

### The lock

`.ergon/init.lock` is JSON and is committed:

```json
{
  "ergon": "1.4.0",
  "files": [
    { "path": ".editorconfig", "producer": "common", "sha256": "4be1…" },
    { "path": ".github/workflows/ci.yml", "producer": "github", "sha256": "0c3a…" },
    { "path": ".golangci.yml", "producer": "go", "local": "9e41…", "sha256": "77d0…" }
  ],
  "answers": {
    "name": "techne",
    "owner": "ThesmOS B.V.",
    "license": "MIT",
    "repository": "dokimasia/techne",
    "security-contact": "security@thesmos.sh",
    "languages": ["go", "typescript"],
    "year": 2026
  }
}
```

The producer of a shared file is the producer of its first fragment. The producer of the fragments of a shared toolchain is the toolchain, such as `jvm`.

| Finding | Condition | `check` | `sync` |
|---|---|---|---|
| `missing` | A file in the lock is absent | Fails | Writes it |
| `edited` | The file's digest differs from the lock | Fails | Reports a conflict and leaves the file, unless `--force` |
| `outdated` | The current rendering differs from the lock, and the file is unedited | Fails | Rewrites it and updates the lock |

`local` is the digest of the file's local file. A changed local file makes its managed file `outdated`.

### Tool versions

Each ergon release embeds the tool versions its baseline was tested with: actions, linters, formatters, test tools, vulnerability scanners, commitlint, and the release of Go that builds commitlint and osv-scanner in CI. Upgrading ergon and running `ergon init sync` moves a repository to them. Dependabot updates dependency manifests and lockfiles, and the action pins in the workflows. A pin it changes in a managed file makes that file `edited`, so `ci.yml`'s `baseline` job fails until the next ergon release contains the same pin.

`baseline.yml` reports a repository whose baseline is behind the newest ergon release. A repository may lag behind the baseline. The issue records that it does.

### Packages

| Package | Contains | Imports |
|---|---|---|
| `core/language/init.go` | The `Initializer` role, `File`, `Class`, `Answers`, `Repository`, `Fixed`, the paths of the shared files, the pinned `Checkout` that every job runs, the runner images, `Go`, the release of Go that a job installs for a tool, and `Job`, which fills these into a job of a language | position 0 |
| `core/language/catalog.go` | The roles of a toolchain and of a language, and `ToolchainRole` and `Role`, which select one | position 0 |
| `ergon-service/baseline` | Rendering, the merge of local files, the lock, and `New`, `Add`, `Remove`, `Check` and `Sync` over the producers that `Open` receives | `core/*` |
| `ergon-service/baseline/common` | The common templates | `core/language` |
| `ergon-service/baseline/github` | The GitHub templates, the pinned actions of the workflows, and the versions of Go and make that they install | `core/language`, `baseline/common` |
| `ergon-lang-<language>/baseline` | The language's templates, its gate tools and their pinned versions, and its fragments | `core/language` |
| `internal/cli` | `ergon init` and its subcommands, which open the repository with the common files and the GitHub files as the producers before the languages | `core/*`, `service/baseline/*` |

The packages for `init` are named `baseline`, because the compiler rejects an import of a package named `init` unless the import renames it. The `baseline` packages of the Java and JavaScript modules also render the fragments of the toolchain that Kotlin and TypeScript share with them.

```go
// Initializer renders the files that a producer contributes to a
// repository. Every language implements it. A toolchain that two languages
// share implements it for the fragments those languages share.
type Initializer interface {
	// Files returns the files and fragments the producer contributes for
	// a, which it reads and does not modify. It reads no file and runs no
	// command, so a rendering depends on a and on the ergon release alone.
	Files(a *Answers) ([]File, error)
}

// File is one file a producer renders.
type File struct {
	// Path is repository-relative and slash-separated.
	Path string

	// Content is the whole rendering of a file the producer writes alone.
	// It is nil when Fragment is set.
	Content []byte

	// Fragment is the producer's part of a shared file. It is nil when
	// Content is set.
	Fragment []byte

	// Class is Managed, Configured or Seeded.
	Class Class
}
```

### Failure handling

| Failure | State afterwards | Recovery |
|---|---|---|
| `new` finds an existing file with other content | Nothing written | Remove the file, or pass `--force` and review the diff |
| `new` finds `.ergon/init.lock` | Nothing written | Use `add` or `sync` |
| `.ergon/init.lock` does not parse | Nothing written | Remove the lock and run `new`, as the error states |
| `.ergon.yaml` does not parse | Nothing written. `check` runs as usual | Correct the YAML error that the message names |
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

**Why not:** several files in the baseline have no such mechanism, among them `.editorconfig`, `.gitignore`, the Makefile and the workflows, so they would still be copies.

### E. A template repository per language

A GitHub template repository per language, copied at creation.

**Why not:** it has no update path at all, and it cannot combine several languages in one repository.

## Drawbacks

- A hand edit to a managed file fails `check`. Every repository-specific setting has to be expressed in a local file.
- A repository receives a baseline change only through an ergon upgrade and a `sync` commit.
- A Dependabot pull request that bumps an action fails `ci.yml`'s `baseline` job until an ergon release contains the same pin.
- ergon maintains templates and pinned tool versions for eleven languages. Each language needs a fixture repository that ergon's CI generates and then gates with that language's own toolchain, nine toolchains in all.
- The merge of a local file has fixed semantics: maps merge, lists append. A setting that needs a list element removed has no expression and needs a change to the baseline.
- The Makefile requires `make` and a POSIX shell on every machine that runs the gate. On Windows these are GNU make and Git Bash.
- The commit-message hook builds commitlint with Go. pre-commit downloads Go on a machine that has none.
- The vulnerability scans query advisory databases over the network, so `make check` fails without network access, in CI and in the pre-commit hook. A new advisory fails the gate of a change that did not touch the affected dependency.
- govulncheck has no option to ignore a finding, so a vulnerability without a fixed release keeps the gate of Go failing.
- The gates of Java and Kotlin need Go to build osv-scanner, and they need a Gradle build that locks its dependencies.
- Each linter runs at the strictest setting that its documentation supports, so a repository that adopts the baseline first fixes or suppresses the findings of its linters.
- The Gradle init script adds PMD's configuration to the build, so a build that locks its dependencies in strict mode writes the lock of that configuration with the init script.
- The Makefile of PHP installs PHPStan and PHP-CS-Fixer with Composer, which checks no checksum of a package.

## Unresolved and future work

- Repository settings on GitHub, such as branch protection, rulesets, environments and required checks, are not proposed.
- Mutation testing gates are not proposed. The dokimi addon supplies them.
- Coverage gates are not proposed. They belong to another component.

## References

| What | Where |
|---|---|
| The earlier ergon's `init` and its templates | `go.thesmos.sh/ergon`, `internal/scaffold/scaffold.go` and `internal/scaffold/templates` |
| copier's update and three-way merge | https://copier.readthedocs.io/en/stable/updating/ |
| The drift measurement | `~/.cache/ergon-init/common.py` and `~/.cache/ergon-init/linters.py`, run on 2026-10-05 |
| The languages that CodeQL analyzes | `src/languages/builtin.json` of github/codeql-action v4.38.2 |
| commitlint | https://github.com/conventionalcommit/commitlint, v0.12.0 |
| The runner images and their software | https://github.com/actions/runner-images |
| GNU make for Windows | https://community.chocolatey.org/packages/make, 4.4.1 |
| shellcheck-py | https://github.com/shellcheck-py/shellcheck-py, 0.11.0.1 |
| govulncheck | https://go.dev/doc/security/vuln/, golang.org/x/vuln v1.8.0 |
| cargo-audit | https://github.com/rustsec/rustsec, cargo-audit 0.22.2 |
| pip-audit | https://github.com/pypa/pip-audit, 2.10.1 |
| osv-scanner | https://github.com/google/osv-scanner, v2.6.0 |
| checkov | https://github.com/bridgecrewio/checkov, 3.3.23 |
| NuGet's audit of packages | https://learn.microsoft.com/nuget/concepts/auditing-packages |
| npm audit and composer audit | https://docs.npmjs.com/cli/commands/npm-audit, https://getcomposer.org/doc/03-cli.md#audit |
| golangci-lint | https://golangci-lint.run, v2.14.0 |
| ruff and mypy | https://docs.astral.sh/ruff/, 0.16.10, and https://mypy.readthedocs.io, 2.4.0 |
| clippy | https://doc.rust-lang.org/clippy/ |
| Biome | https://biomejs.dev, 2.5.15 |
| The analysis of .NET code | https://learn.microsoft.com/dotnet/fundamentals/code-analysis/overview |
| PMD and its base ruleset | https://pmd.github.io, 7.28.0 |
| ktlint | https://pinterest.github.io/ktlint/, 1.8.0 |
| PHPStan and PHP-CS-Fixer | https://phpstan.org, 2.2.17, and https://cs.symfony.com, 3.95.27 |
| tflint | https://github.com/terraform-linters/tflint, v0.64.0 |
