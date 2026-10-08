---
rfc: 0004
title: Repository initialization
author: Roy Klopper
status: Accepted
created: 2026-10-05
updated: 2026-10-08
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0004: Repository initialization

## Summary

`ergon init` sets up a production-grade repository for any combination of eleven languages: C#, Java, Kotlin, PHP, JavaScript, TypeScript, Go, Python, Rust, Terraform and Bash. Each concern of a repository, such as the GitHub files or one language, is a producer. A producer contributes the files of its concern, its section of `.ergon.yaml` and its jobs in CI. It configures nothing outside its concern.

`init` writes the options of every producer to `.ergon.yaml`, where the repository changes them: the versions of the tools and the actions, the runners and the runtime versions of CI, what the targets work on, the steps of the gate and the options of each step. It renders the files it manages from templates embedded in the binary and from those options. The lock, `.ergon/init.lock`, records its answers and the digest of each managed file. The gate runs every tool through `ergon tool run`, which installs the version that `.ergon.yaml` names.

`ergon init check` fails when a managed file has been edited by hand or no longer matches the baseline of the installed ergon. `ergon init sync` brings the files to that baseline.

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
| `ergon init new` | The files of every producer, `.ergon.yaml` and the lock | 1 when a target file exists with other content |
| `ergon init add <language>...` | The language's files, its contributions to the shared files, and its section of `.ergon.yaml` | 1 when a target file exists with other content |
| `ergon init remove <language>...` | Deletes the language's managed files, its contributions and its section of `.ergon.yaml` | 1 when one of those files was edited by hand |
| `ergon init check` | Nothing | 1 when a managed file is missing, edited or outdated |
| `ergon init sync` | Rewrites every outdated managed file that was not edited by hand | 1 when an edited file blocks a rewrite |
| `ergon tool run <section>.<tool>` | The installed tool, in the cache of ergon | The exit status of the tool, as [Tools](#tools) specifies |

`ergon init` without a subcommand prints the subcommands and exits 2. A command line without a required flag also exits 2. `--force` on `new`, `add` and `sync` overwrites edited and existing files. Each command that writes prints a line for every path it wrote or removed. `check` prints a line for every finding, or a JSON array with `--json`. `add` writes the files of a shared toolchain with the first language that names it, and `remove` deletes them with the last.

### Answers

`new` takes the answers as flags and records them in the lock:

| Answer | Flag | Used for |
|---|---|---|
| Name | `--name`, defaulting to the directory's name | `README.md`, `.ergon.yaml`, the fields of a license text |
| Languages | `--language`, repeated | Which producers run |
| Owner | `--owner`, required | The license files, the license section of `.ergon.yaml` |
| License | `--license`, required: one of the 44 identifiers of [RFC-0003](0003-license-headers.md#licenses) | The license files, the license section of `.ergon.yaml` |
| Year | `--year`, defaulting to the current year | The license files |
| Repository | `--repository <owner>/<name>`, required | The GitHub files, the changelog links, the fields of a license text |
| Security contact | `--security-contact`, required | `SECURITY.md`, `CODE_OF_CONDUCT.md` |

`Answers.Validate` checks every answer before a producer renders: a name, an owner and a security contact of one line each, an identifier of `core/spdx`, a year from 1 to 9999, and a repository of the form owner/name. It returns an error that wraps `ErrInvalidAnswer` for the first answer that fails. No producer checks an answer again.

`sync` accepts the same flags but `--language`, and records the answer of each flag in the lock. It renders again every managed file that uses a changed answer, and rewrites the matching keys of `.ergon.yaml`. An answer without a flag keeps its value. `add` and `remove` change the languages.

A key of `.ergon.yaml` that states an answer, such as `license.spdx`, has the answer of the lock, or the earlier answer while `sync` changes it. Every command but `new` fails on any other value, with an error that names the key, the flag of `sync` and the answer. `new` writes its answers over the keys of an existing `.ergon.yaml`.

### File classes

| Class | Written by | Checked |
|---|---|---|
| Managed | `new`, `add`, `sync` | Yes, against the lock |
| Configured | `new`, `add`, `remove`, `sync` | Its options, against each section's options |
| Seeded | `new`, `add`, when the file is absent | No |

- **Managed:** ergon's rendering, line for line. The repository changes it only through the answers, the options of `.ergon.yaml` and its local file.
- **Configured:** `.ergon.yaml`, the configuration of every ergon command. `init` writes the keys its answers determine and a section of options for each producer that has options, as [Settings](#settings) specifies. It changes no other key.
- **Seeded:** a starting text whose content describes the repository, which the repository maintains from then on.

Each managed file that has a comment syntax opens with a comment naming its producer: `Managed by ergon init. Add repository settings to .ergon/local/<path> and run ergon init sync.` When the repository has a local file for the path, the comment names it instead: `Managed by ergon init, with .ergon/local/<path> merged in. Change the repository's settings there and run ergon init sync.` A JSON file and a license text have no comment, and the lock alone records them.

### Producers

| Producer | Files | Section | Contributions to the workflows |
|---|---|---|---|
| `common` | The common files | `common` | The jobs `docs`, `commits` and `changeset` |
| `github` | The GitHub files | `github` | The job `baseline`, and the rendering of every contribution |
| `license` | `LICENSE`, and `NOTICE` for Apache-2.0, as RFC-0003 specifies | `license` | The job `license` |
| `js`, `jvm` | The files that two languages share | `js`, `jvm` | The setup of the jobs of their languages, and the CodeQL analysis and the Dependabot updates of the toolchain |
| Each language | Its own files, and its fragments of the shared files | Its name | The job `check-<language>`, the setup of its toolchain in `release.yml`, and the CodeQL analysis and the Dependabot updates of a language whose toolchain is its own |

`init` renders the common files, the GitHub files and the license files in every repository, a toolchain's files when the first language that names it is chosen, and a language's files when the language is chosen.

A producer configures its own concern alone. Its section names the tools, the actions and the runtime versions of its own producer, and never those of another ecosystem: the section `github` configures the platform, and the section `go` configures the setup of Go in CI.

### Common files

| File | Class | Content |
|---|---|---|
| `.editorconfig` | Managed | Encoding, line endings and indentation, with a section per language |
| `.gitattributes` | Managed | `text=auto eol=lf`, and diff and generated-file rules per language |
| `.gitignore` | Managed | Editor and OS files, and each language's build output and caches |
| `.markdownlint.yml` | Managed | The Markdown rules |
| `.commitlint.yaml` | Managed | The commit-message rules: Conventional Commits types, a subject of at most 72 characters without a final period, and body lines of at most 100 |
| `.pre-commit-config.yaml` | Managed | File hygiene hooks, `make check`, and the commit-message hook, which runs commitlint through `ergon tool run` |
| `Makefile` | Managed | `fmt`, `lint`, `test`, `audit` and `check`, each calling the targets of every language |
| `CODE_OF_CONDUCT.md` | Managed | The Contributor Covenant, with the security contact |
| `.changeset/config.json`, `.changeset/README.md` | Seeded | The release configuration |
| `.ergon.yaml` | Configured | The settings of every ergon command for the chosen languages |
| `README.md`, `CONTRIBUTING.md`, `SECURITY.md` | Seeded | The repository's description, contribution rules and disclosure policy |
| `docs/README.md` and the index of `docs/adr`, `docs/rfc`, `docs/architecture` and `docs/roadmap` | Seeded | The documentation tree |

A repository adds its closed set of commit scopes in its local file of `.commitlint.yaml`, as the rule `scope-enum`.

The job `changeset` runs `ergon release status` against the base commit of each pull request, as RFC-0002 specifies, with the configuration `.changeset/config.json` that the common files seed. It skips the pull requests of Dependabot, as the job `commits` does, and the version pull request from the branch `ergon-release/<base>` of the repository.

### GitHub files

| File | Class | Content |
|---|---|---|
| `.github/workflows/ci.yml` | Managed | The gate on every pull request, on the merge queue and on `main`. The jobs of `release.yml` wait for its run on `main` |
| `.github/workflows/release.yml` | Managed | The release flow of RFC-0002: select-mode, wait, version, pack and publish |
| `.github/workflows/security.yml` | Managed | CodeQL, dependency review and the OpenSSF Scorecard |
| `.github/workflows/codeql.yml` | Managed | The CodeQL analysis of one language, which `security.yml` calls for each language |
| `.github/workflows/baseline.yml` | Managed | A scheduled check against the newest ergon release |
| `.github/actions/setup-ergon/action.yml` | Managed | Installs the ergon release that the lock names on a Linux, macOS or Windows runner, and checks its archive against the release's `checksums.txt` |
| `.github/actions/setup-make/action.yml` | Managed | Installs GNU make from Chocolatey on a Windows runner, whose image has no make |
| `.github/dependabot.yml` | Managed | Weekly updates for the package manager of each language: the minor and patch updates of each directory in one pull request, and each major update in a pull request of its own |
| `.github/ISSUE_TEMPLATE/config.yml`, `bug.yml`, `feature.yml` | Managed | Issue forms, with blank issues off and security reports sent to the advisory form |
| `.github/PULL_REQUEST_TEMPLATE.md` | Managed | The summary, the changeset and the gate checklist |
| `.github/CODEOWNERS` | Seeded | The repository's reviewers |

The workflows:

| Workflow | Triggers | Jobs | Permissions |
|---|---|---|---|
| `ci.yml` | `pull_request`, `merge_group`, `push` to `main` | `check-<language>` for each language, `docs` of the common files, `license` of the license files, `baseline` of the GitHub files, and on pull requests `commits` and `changeset` of the common files | `contents: read` |
| `release.yml` | `push` to `main` | select-mode, wait, version, pack and publish, as RFC-0002 specifies | Per job, as RFC-0002 specifies |
| `security.yml` | `pull_request`, `push` to `main`, weekly | `codeql-<language>` for each CodeQL analysis that a producer contributes, `dependency-review` on pull requests, `scorecard` weekly | `security-events: write` on the CodeQL jobs and `scorecard`, `id-token: write` on `scorecard`, `contents: read` elsewhere |
| `baseline.yml` | Weekly, `workflow_dispatch` | Installs the newest ergon release, runs `ergon init check`, and opens an issue when the baseline is outdated and no such issue is open | `contents: read`, `issues: write` |

`setup-ergon` downloads `ergon_<version>_<os>_<arch>.tar.gz` and `checksums.txt` from the release `v<version>` of github.com/dokimasia/ergon, where `<os>` is `linux`, `darwin` or `windows` and `<arch>` is `amd64` or `arm64`. Every release of ergon publishes these assets. Every job that runs a target of the Makefile installs ergon, because the targets run their tools through `ergon tool run`. The input `version` names another release, or `latest` for the newest one. With the version `source`, the action skips the installation of a release, and a repository that builds ergon from its own source installs it in a step of its local file of the action. ergon's own repository does that, so its gate checks its managed files against the baseline of the same commit. Its local file sets up Go only where the job has not set it up, so a job of Go restores the cache of `actions/setup-go` once.

Every workflow follows these rules:

- The top-level `permissions` is `{}`. Each job grants only the scopes its table row lists.
- Every action is pinned to a full commit SHA, with its release in a comment. The pin is an option of the producer whose job runs the action.
- `actions/checkout` runs with `persist-credentials: false`.
- `concurrency` groups by workflow and ref, and cancels a superseded run on pull requests only.
- Every job has `timeout-minutes`, from the `ci.timeout` of its producer. A job that calls `codeql.yml` takes the timeout of the called job. The job wait of `release.yml` takes the longest timeout of the jobs of `ci.yml` plus the `ci.timeout` of the section `github`, for the time that the jobs of the gate wait for a runner.
- A toolchain's runtime version comes from its pin file, or from the versions that `ci.versions` of its section lists, as a matrix.
- No workflow uses `pull_request_target`, and no pull request job receives a secret.
- Every job specifies its runner image with a version, never a `-latest` label.
- Each job runs on the runners that `ci.runners` of its producer lists. An empty list selects every runner of the section `github`: `ubuntu-26.04`, `macos-26` and `windows-2025` at the baseline. The job of a language runs without `fail-fast`. A job that checks text, such as `docs`, `commits` and `baseline`, and every job of `release.yml` run on the runner of `github.linux`. `baseline` checks text, because the managed `.gitattributes` checks out every text file with LF on every system.
- Each `check-<language>` job runs `make check-<language>`, so CI and a local run execute the same commands. The step runs in `bash`, which is Git Bash on Windows, after `setup-make`.
- A job whose steps run tools keeps the tool directory of ergon in the cache of GitHub Actions, as [Tools](#tools) specifies.

### Jobs

The GitHub producer renders every job of `ci.yml` from one skeleton: the checkout, `setup-make` on a Windows runner, the setup steps of the job's producer, `setup-ergon`, the cache of the tools of ergon for a job that runs tools, and the job's command. The setup steps come before `setup-ergon`, so a repository that builds ergon from its own source builds it with the toolchain that the job set up. A producer returns its jobs through the `Contributor` role as values of `workflow.Job`: the name, the runners, the matrix of runtime versions, the setup steps, the command, the permissions, and whether the command runs tools. It returns its CodeQL analysis, its Dependabot updates and the steps that set up its toolchain in a release the same way. No toolchain appears in the GitHub producer. The GitHub producer rejects a job whose runners its section does not list.

A language's job and its CodeQL analysis run once the repository has the file that pins the language's toolchain:

| Language | Pin file | Setup |
|---|---|---|
| C# | `global.json` | `actions/setup-dotnet`, in the section `csharp` |
| Java, Kotlin | `.java-version` | `actions/setup-java` and Temurin, in the section `jvm` |
| PHP | `.php-version` | `shivammathur/setup-php`, which installs Composer, in the section `php` |
| JavaScript, TypeScript | `package.json` | `actions/setup-node` and `npm ci`, in the section `js` |
| Go | `go.work` | `actions/setup-go`, in the section `go` |
| Python | `pyproject.toml` | None. uv, which `ergon tool run` installs, installs Python |
| Rust | `rust-toolchain.toml` | `actions-rust-lang/setup-rust-toolchain`, in the section `rust` |
| Terraform | `.terraform-version` | `hashicorp/setup-terraform`, in the section `terraform` |
| Bash | none | None |

The job of Go fails in a repository that has a `go.mod` and no `go.work`, because the targets of Go run in the modules of `go.work`. It is skipped in a repository without a `go.mod`.

The jobs version and pack of `release.yml` run the release steps of every producer before `setup-ergon`, each once the repository has the pin file of its toolchain. Go contributes `actions/setup-go` at the version of `go.work`, for the `go mod tidy` of a release.

### Language contributions

Each language produces its own files and its fragments of the shared files, and it contributes to the workflows. A toolchain that two languages share renders the fragments and contributions they share, once, before its first language. The `jvm` toolchain of the Java module renders the target `audit-jvm` and the fragment of `.gitignore` of Gradle, and contributes the setup of Java, the CodeQL analysis of `java-kotlin` and the Gradle updates of Dependabot. The `js` toolchain of the JavaScript module renders `biome.json`, the targets `fmt-js`, `lint-js` and `audit-js`, and the fragments of `.gitattributes` and `.gitignore` that JavaScript and TypeScript share. It contributes the setup of Node.js, the analysis of `javascript-typescript` and the npm updates. The catalog records this rendering as a role of the toolchain, as it records the roles of a language.

`ergon init` renders no build file of a toolchain, such as the Gradle wrapper or a root `package.json`. The repository writes them with its first package.

Each language contributes these parts:

| Shared file | What a language contributes |
|---|---|
| `.gitignore` | A fragment: its build output, caches and tool directories |
| `.editorconfig` | A fragment: a section for its file types |
| `.gitattributes` | A fragment: diff and generated-file rules for its file types |
| `Makefile` | A fragment: `fmt-<language>`, `lint-<language>`, `test-<language>`, `audit-<language>` and `check-<language>`, or `audit-jvm` and `audit-js` for a shared toolchain |
| `.github/workflows/ci.yml` | A `workflow.Job`: the `check-<language>` job, with its toolchain setup |
| `.github/workflows/release.yml` | The `workflow.Step` values that set up its toolchain for a release, such as `actions/setup-go` |
| `.github/workflows/security.yml` | Its CodeQL language, where CodeQL analyzes it: C#, Go, Python and Rust, and the shared `java-kotlin` and `javascript-typescript` |
| `.github/dependabot.yml` | Its package manager: `nuget`, `gradle`, `composer`, `npm`, `gomod`, `uv`, `cargo` or `terraform` |
| `.ergon.yaml` | Its section |

The renderer concatenates the fragments in a fixed order. The common fragment comes first, the GitHub fragment second, and the languages follow in the order of the list in the summary. A shared toolchain comes before its first language. Two renderings of the same answers and local files produce the same bytes. A producer that writes a file that another producer also writes is a defect in ergon. A test over every pair of languages rejects it before release.

`make check` is the repository's single gate.

### Each language's gate

Each language module renders one managed configuration of its linter. Its fragment of the Makefile fills these slots with commands, which run each tool that the toolchain does not include at the version that the language's section of `.ergon.yaml` names:

| Slot | Requirement |
|---|---|
| Formatter | Runs as a check that fails on unformatted files |
| Linter | The strictest ruleset the tool documents, with warnings as errors |
| Type checker or static analyzer | Its strictest mode, where the language has one |
| Tests | Run in CI on every pull request |
| Vulnerability scan | Scans the locked dependencies, or the configuration for Terraform |

Coverage and mutation testing are not part of this gate. The dokimi addon supplies mutation testing for every language.

The fragment of Go also renders `race-go`, which `check-go` runs by default, and `fuzz-go`, `bench-go`, `benchstat-go`, `mutate-go` and `generate-go`, which run on request. `mutate-go` runs dokimi-mutate-go, the Go engine of the dokimi addon. The section `go` of `.ergon.yaml` sets the options of these targets, and its list `check` adds a step to the gate.

- `lint-go` also runs ergon-go-vet in every module, with two analyzers of `ergon-lang-go/analysis`. `errorprefix` reports the text of an error that does not start with the name of its package. `skipexpiry` reports a skipped test whose message names a date that has passed. Both skip generated files, and `go.lint.exclude` names the package patterns they skip, with their external tests. A pattern that `go list` does not resolve fails the run.
- `generate-go` fails when the run changes a tracked file or writes a file that git does not track, so the step `generate` in `check` fails on generated code that is out of date.

The linter configuration of each language:

| Language | Managed file | Linter | Rules |
|---|---|---|---|
| C# | `.globalconfig` | The .NET analyzers, through `dotnet build` and `dotnet format` | Every rule of the latest analysis level as an error, with nullable references and XML documentation |
| Java | `gradle/ergon-java.init.gradle.kts` | PMD 7.28.0, javac and javadoc, through Gradle's `--init-script` | PMD's base ruleset, and every warning of javac and doclint as an error |
| Kotlin | Its section of `.editorconfig` | ktlint 1.8.0 | The official code style of Kotlin, with ktlint's experimental rules |
| PHP | `phpstan.dist.neon` | PHPStan 2.2.17 with phpstan-strict-rules 2.0.12, which the extension installer 1.4.3 of PHPStan loads, and PHP-CS-Fixer 3.95.27 | Level max with every check that the level leaves off, and the PER coding style |
| JavaScript, TypeScript | `biome.json` | Biome 2.5.15, and tsc 7.0.2 for TypeScript | Every stable group of rules as an error, with the project and types domains, and the type-safety options of tsc |
| Go | `.golangci.yml` | golangci-lint v2.14.0, and ergon-go-vet | The 54 linters that a Go repository of the baseline enables, each with the strictest setting that one of them uses, and the two analyzers of ergon-go-vet |
| Python | `ruff.toml` | ruff 0.16.10, and mypy 2.4.0 | Every stable rule of ruff, and mypy's strict mode with twelve more error codes |
| Rust | `clippy.toml` | clippy of the repository's toolchain | Every pedantic lint, and the lints of documentation, as errors |
| Terraform | `.tflint.hcl` | tflint v0.64.0, a release binary | Every rule of the terraform ruleset that tflint bundles |
| Bash | `.shellcheckrc` | shellcheck v0.11.0, a release binary | Eight optional checks |

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
| C# | `audit-csharp` | `dotnet restore --force`, with NuGet's audit of every package and its warnings NU1900 to NU1904 as errors | A known vulnerability at the severity of `csharp.audit.severity` or above, `low` by default, or a vulnerability source that cannot be reached |
| Java, Kotlin | `audit-jvm` | osv-scanner v2.6.0, a release binary, over every `gradle.lockfile` and `buildscript-gradle.lockfile` | A known vulnerability, or a repository without a Gradle lockfile |
| PHP | `audit-php` | `composer audit --locked` | A known vulnerability, or an abandoned package |
| JavaScript, TypeScript | `audit-js` | `npm audit --include=dev` | A known vulnerability at the severity of `js.audit.severity` or above, `low` by default |
| Go | `audit-go` | govulncheck v1.8.0, once per module of `go.work` | A known vulnerability that the code calls |
| Python | `audit-python` | pip-audit 2.10.1 through uv, over the packages of `uv.lock` that `uv export` writes to a `pylock.toml` | A known vulnerability, or a package that it cannot check |
| Rust | `audit-rust` | cargo-audit 0.22.2, which `cargo install --locked` installs | A known vulnerability |
| Terraform | `audit-terraform` | checkov 3.3.23 through uv, on Python 3.13 | A failed check, or a file that does not parse |
| Bash | none | Bash has no locked dependencies | |

The jvm toolchain renders `audit-jvm`, and the js toolchain renders `audit-js`, once for both of their languages. A language of a shared toolchain names the target of its step `audit` after itself, such as `audit-java`, and the target requires the target of the toolchain. A Java or Kotlin repository locks the dependencies of its Gradle build, because Gradle writes a lockfile only for a build with dependency locking, and ergon init renders no build file that enables it.

pip-audit scans the export of `uv.lock`, because `uv audit` is a preview command whose interface can change in any uv release. checkov scans the configuration of Terraform, because Trivy, the other scanner that covers it, takes 89 s to build with `go run` and leaves a 2.6 GB build cache, against 4 s for checkov through uv.

### Templates

A producer's templates are a file tree that mirrors the repository. The class of a file follows from the directory of its template, so no producer lists its files:

- `managed/<path>.tmpl` renders a managed file.
- `seeded/<path>.tmpl` renders a seeded file.
- `shared/<path>.tmpl` renders the producer's fragment of a shared file.

One engine in `service/baseline/render` executes the templates of every producer with `text/template`:

- The delimiters are `{{%` and `%}}`, so the `${{ }}` expressions of a workflow and the `{{.Dir}}` of `go list -f` in a Makefile remain text.
- A template reads `.Answers`, `.Options`, which are the options of its producer, `.Data`, which are the values that its producer computes, and `.Contributions`, which are the contributions of every producer to the workflows. A key that the data lacks is an error.
- The function `words` writes a list as words of the shell, escaped for make. `make` escapes a value for make, `yaml` quotes a scalar of YAML, and `steps` writes the steps of a job of a workflow, so `ci.yml` and `release.yml` write a step the same way.
- The engine skips a template that renders zero bytes, as `NOTICE` does for every license but Apache-2.0.

A producer whose options state the paths of its files renders those files through the role `Placer`, beside its templates. The license producer renders the `LICENSE` of each directory of `license.directories` this way, as RFC-0003 specifies. The engine adds each placed file to the rendering as a managed file of its producer. It rejects a path that is not relative, clean and slash-separated, a path under `.ergon/`, and a path that another file of the rendering has.

A test renders every template of every producer with its options at the baseline. It checks that each file parses in its format, that no line ends in whitespace, and that each file ends with one newline, which the hooks of `.pre-commit-config.yaml` require.

### Settings

`.ergon.yaml` has a section for each producer that has options: `common`, `github`, `license`, each language, and each toolchain that two languages share, `js` and `jvm`. Each producer declares its options as a struct in its own package. The struct composes the option types of `core/option`, so a key means the same in every section, and adds the options that only its producer has.

| Key | Type | Meaning |
|---|---|---|
| `tools` | One tool of `core/option` per entry | The tools that the producer's targets run, by the tool's own name |
| `paths` | A list of paths | What the targets work on: the package patterns of each Go module, or the paths for the other languages |
| `check` | A list of steps | The steps that `check-<section>` runs, in order |
| `test`, `race`, `generate` | `Run` | The options of the step |
| `fuzz` | `Fuzz` | The options of the step |
| `bench` | `Bench` | The options of the step |
| `mutate` | `Mutate` | The options of the step |
| `audit` | `Audit` | The options of the step |
| `ci` | `CI` | `actions`, the pins of the actions of the producer's jobs, and `timeout`. The section of a toolchain adds `runners` and `versions` |

A step is a target of the Makefile: the step `test` of the section `python` is `test-python`. The steps are `fmt`, `lint`, `test`, `race`, `fuzz`, `bench`, `mutate`, `generate` and `audit`, and a section has a key only for a step that its producer has. A language whose toolchain is shared runs the toolchain's target where it has none of its own, so `audit` in `typescript` runs `audit-js`. `fmt` takes no options, because the managed configuration of the tool states its rules. `lint` takes an option only for a tool without a configuration file. `go.lint.exclude`, for ergon-go-vet, is the one such option.

| Option | Steps | Meaning |
|---|---|---|
| `args` | `test`, `race`, `fuzz`, `bench`, `mutate`, `generate`, `audit` | The arguments that follow the step's command, each on one line |
| `match` | `fuzz`, `bench` | A regular expression of the names of the fuzz targets or the benchmarks that run, on one line and without a single quote |
| `time` | `fuzz`, `bench` | The time of each fuzz target, or of each run of a benchmark: a positive duration, or a positive number of iterations such as `100x` |
| `count` | `bench` | The runs of each benchmark, at least 1 |
| `workers` | `mutate` | The mutants that run at once, at least 1 |
| `timeout` | `mutate` | The limit of the whole run, as a duration. `0s` sets none |
| `ignore` | `audit` | The advisories that the scan accepts, by identifier, for pip-audit, cargo-audit and checkov |
| `severity` | `audit` | The lowest severity that fails the scan, `low`, `moderate`, `high` or `critical`, for npm audit and NuGet |

The keys outside the steps accept these values:

- `paths`: paths of one line each, none empty.
- `check`: steps that the producer runs.
- `ci.actions`: an action of the form `owner/name` or `owner/name/path`, a commit of 40 hexadecimal digits, and a release.
- `ci.runners`: runner images of the section `github`. An empty list selects every runner of that section.
- `ci.versions`: the runtime versions of the toolchain. An empty list selects the version of the pin file.
- `ci.timeout`: the minutes of each job, at least 1.

A tool has one of these kinds, which the type of its field states:

| Kind | Value | Runs as |
|---|---|---|
| Go module | `<module>@<version>` | `go install` into the cache, in the gate of Go alone |
| PyPI package | `<package>@<version>` | `uv tool run`, or `uv run --with` in the project's environment, through the uv of its section |
| npm package | `<package>@<version>` | `npx`, in the gates of JavaScript and TypeScript alone |
| Crate | `<crate>@<version>` | `cargo install --locked`, in the gate of Rust alone |
| Maven artifact | `<group>:<artifact>@<version>` | Gradle, or `java -jar` after a check against the `.sha256` file of Maven Central, in the gates of Java and Kotlin alone |
| Composer package | `<vendor>/<package>@<version>` | Composer, in the gate of PHP alone |
| Release binary | `version`, and the SHA-256 digest of the asset of each platform | Directly, after a check of the digest |

A package and a version are neither empty, and contain letters, digits, `.`, `_`, `-` and `+`. A package may also contain `/`, `:` and `@`. The Makefile writes a path or an argument as a word of the shell, and quotes a word that has a character that the shell reads.

Each field of an options struct has a `doc` tag. `init` writes the tag as the comment above the key, so the struct is the one source of the key, its baseline value and its meaning. The struct's `Validate` checks the rules of its producer that a type of `core/option` cannot state, such as a step of `check` that the producer has, a `fuzz.match` that compiles as a regular expression of Go, and the parameters of BUSL-1.1.

The sections and the baseline value of each option, in the order in which `init new` writes them for every language, with `<sha256>` for each digest:

```yaml
common:
  tools:
    commitlint:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 0.12.0
  pre-commit-hooks: v6.0.0
  ci:
    actions:
      markdownlint:
        uses: DavidAnson/markdownlint-cli2-action
        commit: 21c1be1b93ad9ed58fa840aacc3f279cde2a72ff
        release: v24.2.0
    timeout: 10
github:
  runners: [ubuntu-26.04, macos-26, windows-2025]
  linux: ubuntu-26.04
  make: 4.4.1
  ci:
    actions:
      checkout:
        uses: actions/checkout
        commit: 3d3c42e5aac5ba805825da76410c181273ba90b1
        release: v7.0.1
      codeql:
        uses: github/codeql-action
        commit: 2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2
        release: v4.38.2
      dependency-review:
        uses: actions/dependency-review-action
        commit: a1d282b36b6f3519aa1f3fc636f609c47dddb294
        release: v5.0.0
      scorecard:
        uses: ossf/scorecard-action
        commit: 2d1146689b8cda280b9bc96326124645441f03bc
        release: v2.4.4
      upload-artifact:
        uses: actions/upload-artifact
        commit: cf430e030ddbb5b0abf93d22962f4752f3646cd9
        release: v7.0.2
      download-artifact:
        uses: actions/download-artifact
        commit: 9000827ccba6bdab643e8b6fd33ac0654aef8333
        release: v8.0.2
      cache:
        uses: actions/cache
        commit: 55cc8345863c7cc4c66a329aec7e433d2d1c52a9
        release: v6.1.0
    timeout: 15
license:
  owner: Example B.V.
  spdx: MIT
  parameters:
    licensed-work: ""
    additional-use-grant: ""
    change-date: ""
    change-license: ""
  directories: []
  styles: {}
  exclude: []
  ci:
    actions: {}
    timeout: 10
csharp:
  check: [lint, test, audit]
  test:
    args: []
  audit:
    severity: low
  ci:
    actions:
      setup-dotnet:
        uses: actions/setup-dotnet
        commit: a98b56852c35b8e3190ac28c8c2271da59106c68
        release: v6.0.0
    runners: []
    versions: []
    timeout: 30
jvm:
  tools:
    osv-scanner:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 2.6.0
  ci:
    actions:
      setup-java:
        uses: actions/setup-java
        commit: de7274f081f381c8f8158605e0321c36c376e2e6
        release: v6.0.1
    runners: []
    versions: []
    timeout: 30
java:
  tools:
    pmd: net.sourceforge.pmd:pmd-java@7.28.0
  check: [lint, test, audit]
  test:
    args: []
kotlin:
  tools:
    ktlint: com.pinterest.ktlint:ktlint-cli@1.8.0
  check: [lint, test, audit]
  test:
    args: []
php:
  tools:
    phpstan: phpstan/phpstan@2.2.17
    phpstan-strict-rules: phpstan/phpstan-strict-rules@2.0.12
    phpstan-extension-installer: phpstan/extension-installer@1.4.3
    php-cs-fixer: php-cs-fixer/shim@3.95.27
  paths: [.]
  check: [lint, test, audit]
  test:
    args: []
  ci:
    actions:
      setup-php:
        uses: shivammathur/setup-php
        commit: f3e473d116dcccaddc5834248c87452386958240
        release: 2.37.2
    runners: []
    versions: []
    timeout: 30
js:
  tools:
    biome: '@biomejs/biome@2.5.15'
  paths: [.]
  audit:
    severity: low
  ci:
    actions:
      setup-node:
        uses: actions/setup-node
        commit: 820762786026740c76f36085b0efc47a31fe5020
        release: v7.0.0
    runners: []
    versions: []
    timeout: 30
javascript:
  check: [lint, test, audit]
  test:
    args: []
typescript:
  tools:
    typescript: typescript@7.0.2
  check: [lint, test, audit]
  test:
    args: []
go:
  tools:
    golangci-lint: github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
    govulncheck: golang.org/x/vuln/cmd/govulncheck@v1.8.0
    benchstat: golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68
    dokimi-mutate-go: go.dokimi.dev/mutate/cmd/dokimi-mutate-go@v0.0.0-20261006212535-719083ce3457
    ergon-go-vet: go.dokimi.dev/ergon/lang/go/cmd/ergon-go-vet@<the release of ergon-lang-go>
  paths: [./...]
  check: [lint, test, race, audit]
  lint:
    exclude: []
  test:
    args: [-count=1]
  race:
    args: [-count=1, -p=1]
  fuzz:
    match: .
    time: 30s
    args: [-fuzzminimizetime=5s]
  bench:
    match: .
    time: 1s
    args: [-benchmem]
    count: 6
  mutate:
    timeout: 0s
    args: []
    workers: 1
  generate:
    args: []
  audit:
    args: []
  ci:
    actions:
      setup-go:
        uses: actions/setup-go
        commit: b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
        release: v7.0.0
    runners: []
    versions: []
    timeout: 30
python:
  tools:
    uv:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 0.12.23
    ruff: ruff@0.16.10
    mypy: mypy@2.4.0
    pytest: pytest@9.1.1
    pip-audit: pip-audit@2.10.1
  paths: [.]
  check: [lint, test, audit]
  test:
    args: []
  audit:
    ignore: []
  ci:
    actions: {}
    runners: []
    versions: []
    timeout: 30
rust:
  tools:
    cargo-audit: cargo-audit@0.22.2
  check: [lint, test, audit]
  test:
    args: [--all-targets]
  audit:
    ignore: []
  ci:
    actions:
      setup-rust-toolchain:
        uses: actions-rust-lang/setup-rust-toolchain
        commit: ecabd13d1c56bd1345c230e542e9144811ad706f
        release: v2.0.0
    runners: []
    versions: []
    timeout: 30
terraform:
  tools:
    tflint:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 0.64.0
    uv:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 0.12.23
    checkov: checkov@3.3.23
  paths: [.]
  check: [lint, test, audit]
  test:
    args: []
  audit:
    ignore: []
  ci:
    actions:
      setup-terraform:
        uses: hashicorp/setup-terraform
        commit: dfe3c3f87815947d99a8997f908cb6525fc44e9e
        release: v4.0.1
    runners: []
    versions: []
    timeout: 30
bash:
  tools:
    shellcheck:
      sha256:
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        windows/amd64: <sha256>
      version: 0.11.0
  paths: ['*.sh', '*.bash']
  check: [lint]
  ci:
    actions: {}
    runners: []
    timeout: 30
```

The section of every toolchain has a `ci` key with the setup action of its own toolchain: `actions/setup-dotnet` in `csharp`, `actions/setup-java` in `jvm`, `shivammathur/setup-php` in `php`, `actions/setup-node` in `js`, `actions/setup-go` in `go`, `actions-rust-lang/setup-rust-toolchain` in `rust` and `hashicorp/setup-terraform` in `terraform`. The sections `python` and `bash` have no setup action. Bash has no runtime version, so its `ci` has no `versions`. The sections `java`, `kotlin`, `javascript` and `typescript` have no `ci` key, because the section of their toolchain has it. `init` writes `license.owner` and `license.spdx` from its answers, as RFC-0003 specifies.

The life of an option:

- `new` and `add` write the section of each producer with every option at its baseline value, under the comment of its `doc` tag. `remove` deletes the section of each producer that it removes.
- The lock records the baseline value of every option that `init` wrote. `sync` replaces an option whose value equals the record with the baseline value of the installed ergon, and keeps an option that the repository changed. An option that the repository deleted returns at its baseline value.
- `sync` renders the managed files that state an option from the options of the sections. A changed option makes those files `outdated`.
- `check` and every command that writes decode each section strictly into the struct of its producer, and run its `Validate`. They reject a key that the struct does not have, a section of a producer that the repository does not have, and a value that the producer does not accept. The error message contains the key.
- The Makefile assigns each option of a step to a variable named after its key, such as `GO_FUZZ_MATCH`, with `?=`, so one run overrides it on the command line: `make fuzz-go GO_FUZZ_MATCH=FuzzDecode`.
- The Makefile refers to a tool by its name, as `$(ERGON) tool run go.golangci-lint`. The version of each tool appears once, in `.ergon.yaml`, and `make` does not read YAML.
- The local file of the Makefile adds the repository's own targets. It sets no option.

A new version of a release binary needs the digest of each platform's asset in `.ergon.yaml`. A package of a registry needs none. The program that installs the package checks what it downloads:

- `go install` checks a Go module against the checksum database.
- uv, npm and cargo check a package against the digest that their registry publishes.
- ergon checks the jar of a Maven artifact against the `.sha256` file of Maven Central.
- Composer checks no digest. The drawbacks list this.

A tool whose integration names its package accepts another version and rejects another package:

- `java.tools.pmd`, because Gradle's PMD plugin runs `net.sourceforge.pmd:pmd-java`.
- `kotlin.tools.ktlint`, because `ergon tool run` runs the jar of `com.pinterest.ktlint:ktlint-cli` with the classifier `all`, which contains ktlint and its dependencies.
- The four tools of `php`, because the targets run the programs of PHPStan and PHP-CS-Fixer with the flags of those programs, and the extension installer of PHPStan loads phpstan-strict-rules.

The actions of the workflows are options of the producers whose jobs run them: the checkout, the cache of the tools of ergon, the actions of `security.yml` and the artifact actions of `release.yml` in `github`, markdownlint in `common`, and the setup of each toolchain in its section. Dependabot cannot update `.ergon.yaml`, so `dependabot.yml` updates no action.

Only Go has the steps `race`, `fuzz`, `bench`, `mutate` and `generate`. A fuzzer or a benchmark harness in another language is a dependency of the repository's own build, such as criterion, JMH or pytest-benchmark, and `init` renders no build file. A language gains `mutate` with its engine of the dokimi addon, and dokimi-mutate-go is the only one.

### Tools

`ergon tool run <section>.<tool> [-- <arguments>]` installs the tool that the section names and runs it with the arguments. Every target of the Makefile and every hook runs its tools this way, so a gate needs no runtime of another ecosystem.

- `ergon tool run` resolves the options of the section as `init` does, from `.ergon.yaml`, the lock and the baseline of the installed ergon, in the repository of the working directory or of the nearest of its parents with `.ergon/init.lock`.
- It runs the tool in the working directory, so a target that runs in each module of Go runs the tool in the module.
- It installs a tool once into `ergon/tools` in the user's cache directory, under the tool's kind, name, version and platform, with the digest of a release binary, and reuses it.
- It checks the asset of a release binary against the SHA-256 digest of the platform before it unpacks the asset. The producer derives the URL of the asset and the path of the program in it from the version and the platform. An asset is an archive, or the program itself, as osv-scanner publishes it.
- A PyPI package runs through the release binary `uv` of the same section.
- A Go module, an npm package, a crate, a Maven artifact and a Composer package run through the toolchain of their own gate, which the job of that gate sets up.
- The Composer packages of a section install together into one project of their own, which allows the plugins of its packages, so the extension installer of PHPStan loads phpstan-strict-rules.
- The command exits with the exit status of the tool. It exits 1 with an error when the tool cannot be installed, and 2 for a section of no producer of the repository or a tool that the section does not name.

The release binaries of the baseline are commitlint v0.12.0 in `common`, osv-scanner v2.6.0 in `jvm`, tflint v0.64.0 in `terraform`, shellcheck v0.11.0 in `bash`, and uv 0.12.23 in `python` and `terraform`. Each release publishes an asset for linux/amd64, darwin/arm64 and windows/amd64. The releases of commitlint, osv-scanner, tflint and uv publish a checksum file beside the assets. The release of shellcheck publishes none, so the baseline pins the digests of the assets themselves.

A job of `ci.yml` whose steps run tools restores the tool directory of ergon from the cache of GitHub Actions before its steps, with `actions/cache`, and saves the directory after a run that succeeds:

- The directory is `ergon/tools` in the cache directory of the user: `~/.cache` on Linux, `~/Library/Caches` on macOS and `~/AppData/Local` on Windows.
- The key is the system, the architecture, the job, the runtime version of the job's matrix, and the digest of `.ergon.yaml` and `.ergon/init.lock`. A changed version of a tool or of ergon changes the key, so the job installs its tools again.
- A producer marks each job whose steps run a tool that `ergon tool run` installs into the directory. These are the job `commits` and the check jobs of Go, Python, Rust, Terraform, Bash, PHP, Java and Kotlin. JavaScript and TypeScript run their tools through npx, which keeps its own cache, and C# runs none.

Without the cache, the first lint of each Go job of ergon's own repository built golangci-lint from 203 downloaded modules on 2026-10-08. That lint took 57 s on Linux, 69 s on macOS and 104 s on Windows, against 4 to 15 s for the lint of each other module.

### Several languages in one repository

The workspace file of each language is at the repository root: `go.work`, `package.json`, `pyproject.toml`, `Cargo.toml`, `settings.gradle.kts`, `Directory.Build.props`, `composer.json`. Their names differ, so they coexist. Java and Kotlin share the Gradle build of `settings.gradle.kts`. A language's packages are in directories the repository chooses. The targets of Go lint and test every module that `go list -m` lists, which is every module of `go.work`.

`ergon init` writes no source code. A language's gate runs once the repository has its first package.

### Local files

A repository adds its own content to a managed file through `.ergon/local/<path>`, such as its commit scopes in `.ergon/local/.commitlint.yaml` or its own targets in `.ergon/local/Makefile`. The tools and the options of the steps are options of `.ergon.yaml`, and no local file sets them:

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
  "settings": {
    "go.check": ["lint", "test", "race", "audit"],
    "go.fuzz.time": "30s",
    "go.tools.golangci-lint": "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0"
  },
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

The producer of a shared file is the producer of its first fragment. The producer of the fragments of a shared toolchain is the toolchain, such as `jvm`. `settings` records the baseline value of each option that `init` wrote, by its key in `.ergon.yaml`.

| Finding | Condition | `check` | `sync` |
|---|---|---|---|
| `missing` | A file in the lock is absent | Fails | Writes it |
| `edited` | The file's digest differs from the lock | Fails | Reports a conflict and leaves the file, unless `--force` |
| `outdated` | The current rendering differs from the lock, and the file is unedited | Fails | Rewrites it and updates the lock |

`local` is the digest of the file's local file. A changed local file makes its managed file `outdated`.

`ergon` is the release of ergon that wrote the lock, or `dev` for a build without a release. A build of `go install go.dokimi.dev/ergon/cmd/ergon@v1.4.0` writes `1.4.0`.

### Tool versions

Each ergon release embeds the baseline value of every option, among them the version of each tool its baseline was tested with, the digests of its release binaries, and the pin of each action. Upgrading ergon and running `ergon init sync` moves every option that still has the earlier baseline value to the new one. A repository chooses another version in its section of `.ergon.yaml`. Dependabot updates the dependency manifests and the lockfiles.

`baseline.yml` reports a repository whose baseline is behind the newest ergon release. A repository may lag behind the baseline. The issue records that it does.

### Packages

| Package | Contains | Imports |
|---|---|---|
| `core/language/init.go` | The roles `Producer`, `Calculator`, `Configurable`, `Contributor` and `Placer`, `File`, `Options`, `Answers` with `Validate`, `Repository`, and the path of `.ergon.yaml` | `core/spdx`, `core/workflow`, `core/workspace` |
| `core/language/catalog.go` | The roles of a toolchain and of a language, and `ToolchainRole` and `Role`, which select one | `core/workspace` |
| `core/option` | The kinds of tool, with `Binary`, `Release` and `UV`, the step options `Run`, `Fuzz`, `Bench`, `Mutate`, `Audit` and `Threshold`, `Check`, `Step`, `Severity`, `Paths`, `Version`, and `CI` with its forms for the runners and the versions, each with `Validate` | `core/workflow` |
| `core/workflow` | `Job`, `Setup`, `Step`, `Action`, `CodeQL`, `Update` and `Contribution`: a producer's part of the workflows | the standard library |
| `core/spdx` | The identifiers of the licenses, as RFC-0003 specifies | the standard library |
| `ergon-service/baseline` | `New`, `Add`, `Remove`, `Check`, `Sync` and `Options` over the producers that `Open` receives, and the plan of their changes | `core/*`, `baseline/lock`, `baseline/options`, `baseline/overlay`, `baseline/render` |
| `ergon-service/baseline/lock` | The format of `.ergon/init.lock` | `core/*` |
| `ergon-service/baseline/options` | The sections of `.ergon.yaml`: the resolution against the record and the answers of the lock, the strict decode into the struct of a producer, its `Validate`, and the sections with the comments of the `doc` tags | `core/*`, viper and mapstructure for the decode, go.yaml.in/yaml/v3 for the writer |
| `ergon-service/baseline/overlay` | The local files: the merge of YAML and the appended text | `core/*`, go.yaml.in/yaml/v3 |
| `ergon-service/baseline/render` | The engine of the templates, the classes from the template tree, the files of a `Placer`, the collection of the contributions, and the join of fragments | `core/*` |
| `ergon-service/baseline/common` | The producer of the common files | `core/*`, `service/release` for the branch of the version pull request |
| `ergon-service/baseline/github` | The producer of the GitHub files, which renders the contributions of every producer | `core/*` |
| `ergon-service/licenses/baseline` | The producer of the license files, as RFC-0003 specifies | `core/*`, `service/licenses` |
| `ergon-service/tool` | `ergon tool run`: the installation, the check and the run of a tool | `core/language`, `core/option` |
| `ergon-lang-<language>/baseline` | The producer of the language: the struct of its options with their rules, its templates, and its contributions | `core/*` |
| `ergon-lang-go/analysis` | The analyzers `errorprefix` and `skipexpiry` | `golang.org/x/tools/go/analysis` |
| `ergon-lang-go/cmd/ergon-go-vet` | The command that runs the analyzers of `analysis` | `ergon-lang-go/analysis`, `golang.org/x/tools/go/analysis/multichecker`, `golang.org/x/tools/go/packages` |
| `internal/cli` | `ergon init` and its subcommands, `ergon license`, `ergon release` and `ergon tool run`, which open the repository with the common files, the GitHub files and the license files as the producers before the languages | `core/*`, `service/*` |

The packages for `init` are named `baseline`, because the compiler rejects an import of a package named `init` unless the import renames it. The `baseline` packages of the Java and JavaScript modules also render the fragments and the contributions of the toolchain that Kotlin and TypeScript share with them.

```go
// Producer renders the files of one concern of a repository: the common
// files, the GitHub files, the license files, a toolchain that two
// languages share, or a language.
type Producer interface {
	// Templates returns the templates of the producer. Their tree mirrors
	// the repository under managed/, seeded/ and shared/, and each name
	// ends in .tmpl.
	Templates() fs.FS
}

// Calculator is a producer whose templates read values that it computes.
type Calculator interface {
	// Data returns the values that the templates read as .Data, from the
	// answers, the options of the producer and the contributions of every
	// producer. It reads no file and runs no command, so a rendering
	// depends on a, o, c and the ergon release alone.
	Data(a *Answers, o Options, c *workflow.Contribution) (any, error)
}

// Options are the options of a producer's section of .ergon.yaml: a
// pointer to a struct whose fields have yaml and doc tags.
type Options interface {
	// Validate returns an error for the first value that the producer
	// cannot render.
	Validate() error
}

// Configurable is a producer with a section of .ergon.yaml.
type Configurable interface {
	// Options returns a new value of the producer's options at the
	// baseline.
	Options() Options
}

// Contributor is a producer with a part of the workflows: jobs of ci.yml,
// the setup of its toolchain in release.yml, a CodeQL analysis, or a
// package manager that Dependabot updates.
type Contributor interface {
	// Contribution returns the producer's part of the workflows for o.
	Contribution(o Options) workflow.Contribution
}

// Placer is a producer that renders managed files at paths that its
// options state, beside the files of its templates.
type Placer interface {
	// Files returns the files of the producer for the answers a, its
	// options o and the contributions c. It reads no file and runs no
	// command, as Data does.
	Files(a *Answers, o Options, c *workflow.Contribution) ([]File, error)
}

// File is a managed file that a Placer renders.
type File struct {
	// Path is the path of the file in the repository: relative, clean and
	// slash-separated.
	Path string

	// Content is the content of the file.
	Content []byte
}
```

### Failure handling

| Failure | State afterwards | Recovery |
|---|---|---|
| `new` finds an existing file with other content | Nothing written | Remove the file, or pass `--force` and review the diff |
| `new` finds `.ergon/init.lock` | Nothing written | Use `add` or `sync` |
| `.ergon/init.lock` does not parse | Nothing written | Remove the lock and run `new`, as the error states |
| `.ergon.yaml` does not parse | Nothing written, and `check` fails | Correct the YAML error that the message names |
| An option that its section does not have, or a value that the producer does not accept | Nothing written, and `check` fails | Correct the key that the error names |
| A key of an answer, such as `license.spdx`, that differs from the answer of the lock | Nothing written, and every command that reads `.ergon.yaml` fails | Run `sync` with the flag that the error names, or restore the answer |
| An answer that `Answers.Validate` rejects | Nothing written | Pass the flag again with a value that the error describes |
| An unknown language | Nothing written | The error lists the eleven languages |
| A local file for a path that is not managed | Nothing written | The error names the local file |
| `sync` meets an edited file | Every other outdated file is rewritten. The edited file is unchanged | Move the edit into the local file, then run `sync --force` |
| `remove` meets an edited file | Nothing removed | The same |
| A template reads a key that its data does not have | Nothing written | A defect of ergon, which the test of every template prevents |
| A producer places a file at a path that is not relative and clean, under `.ergon/`, or of another file | Nothing written | A defect of ergon, which the tests of the producer prevent |
| `ergon tool run` names a section or a tool that the repository does not have | Nothing run, exit 2 | The error lists the tools of the section |
| `ergon tool run` in a directory without `.ergon/init.lock` in it or a parent | Nothing run, exit 1 | Run the command in the repository, or run `ergon init new` |
| A release binary has no digest for the platform | Nothing installed, exit 1 | Add the digest of the platform's asset to the section |
| A download differs from its digest | Nothing installed, exit 1 | Check the version and the digests of the tool |
| The toolchain of a tool's kind is missing, such as `go` for a Go module | Nothing installed, exit 1 | Run the target in the gate of its toolchain, or install the toolchain |

Every command computes its whole change before it writes. It writes each file to a temporary name in the same directory and renames it, so an interrupted run leaves each file either old or new.

### Adopting an existing repository

`ergon init new --force` in a repository that predates ergon's baseline overwrites every managed file. The diff then shows each repository-specific setting the baseline lacks. Each such setting moves into a local file before the commit. `init` keeps every key of `.ergon.yaml` that is not the section of a producer, so the keys of the earlier ergon, such as `checks` and `bootstrap`, remain until the adoption deletes them.

The earlier ergon's `license` section is the exception. Its name is the section of the license producer, whose struct has none of its keys, so `new` fails until the adoption deletes the section, as RFC-0003 specifies. On 2026-10-07 `new --force` adopted clones of techne, treesitter, assert-python, assert-typescript, assert-java and assert-rust this way.

## Alternatives considered

### A. Generate once

The earlier ergon wrote the files once and left them to the repository.

**Why not:** the 18 repositories it set up have up to 14 distinct versions of one file. Standards such as `paralleltest` and 100% coverage are enforced in only some of them. One-shot copies stop receiving improvements, and nothing reports it.

### B. Three-way merge updates

copier records the template version and the answers. `copier update` merges the newer template into the edited files.

**Why not:** a merge keeps every hand edit, including edits that weaken a gate, and nothing reports the difference from the standard. A conflict leaves markers in a configuration file that a person resolves by hand, in every repository at every update.

### C. Answers in `.ergon.yaml`

`init` would read its languages, owner and repository from `.ergon.yaml`, as it reads the options.

**Why not:** the commands themselves change the answers: `add` and `remove` change the languages, and `sync` changes an answer that a flag names. A hand edit of an answer in `.ergon.yaml` would contradict the lock. The answers belong to the lock, and the options to `.ergon.yaml`.

### D. Shared configuration through each tool's `extends` alone

Each repository would reference a published base configuration instead of a copy.

**Why not:** several files in the baseline have no such mechanism, among them `.editorconfig`, `.gitignore`, the Makefile and the workflows, so they would still be copies.

### E. A template repository per language

A GitHub template repository per language, copied at creation.

**Why not:** it has no update path at all, and it cannot combine several languages in one repository.

### F. One options struct for every language

`core` would declare one struct with every option of every step, and the service would validate the options of every language by reflection over it.

**Why not:** an option of one language, such as the advisory identifiers of pip-audit, would change `core` and the service, and no language could validate a value by its own rules.

### G. Each tool through the installer of its ecosystem

Every gate would install a tool as its ecosystem does: a Go program with `go run`, a Python program with uv.

**Why not:** three Go programs run outside the gate of Go, commitlint, osv-scanner and tflint, and two Python programs outside the gate of Python, checkov and shellcheck-py. Each of their jobs would install Go or uv, and the section of a producer would configure another ecosystem.

### H. Actions that Dependabot updates

The workflows would pin each action as a constant of the ergon release, and Dependabot would propose each new release.

**Why not:** a pin that Dependabot changes makes the managed workflow `edited`, and a repository could not choose another release of an action in `.ergon.yaml`.

## Drawbacks

- A hand edit to a managed file fails `check`. Every repository-specific setting has to be expressed in `.ergon.yaml` or in a local file.
- An option that has the baseline value follows the baseline, so a repository cannot pin an option to the current baseline value across an upgrade of ergon.
- A tool version that a repository chooses is one that no ergon release tested.
- A repository receives a baseline change only through an ergon upgrade and a `sync` commit.
- Dependabot proposes no release of an action. A repository receives a new release of an action through an ergon release, or through a pin that it sets in its section.
- ergon maintains templates and pinned tool versions for eleven languages. Each language needs a fixture repository that ergon's CI generates and then gates with that language's own toolchain, nine toolchains in all.
- The merge of a local file has fixed semantics: maps merge, lists append. A setting that needs a list element removed has no expression and needs a change to the baseline.
- The Makefile requires `make` and a POSIX shell on every machine that runs the gate. On Windows these are GNU make and Git Bash.
- `make` needs ergon on the PATH, because every target runs its tools through `ergon tool run`.
- A new version of a release binary needs the digest of each platform's asset in `.ergon.yaml`.
- The first run of a tool on a machine downloads it, so the first gate needs the network for the tools as well as for the advisories.
- The vulnerability scans query advisory databases over the network, so `make check` fails without network access, in CI and in the pre-commit hook. A new advisory fails the gate of a change that did not touch the affected dependency.
- govulncheck has no option to ignore a finding, so a vulnerability without a fixed release keeps the gate of Go failing.
- The gates of Java and Kotlin need a Gradle build that locks its dependencies.
- Each linter runs at the strictest setting that its documentation supports, so a repository that adopts the baseline first fixes or suppresses the findings of its linters.
- The Gradle init script adds PMD's configuration to the build, so a build that locks its dependencies in strict mode writes the lock of that configuration with the init script.
- `ergon tool run` installs the Composer packages of PHP, which Composer checks against no checksum, into a project that allows the plugins of its packages.
- A template that reads a key its data lacks fails when it renders, not when it compiles. The test that renders every template of every producer is the check.
- The cache of CI keeps a tool that a toolchain built, such as a Go module that `go install` built, until a version of a tool or of ergon changes. A later minor release of Go in `go.work` then runs a golangci-lint that the earlier release built, and golangci-lint refuses a Go version newer than the one that built it.

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
| shellcheck | https://github.com/koalaman/shellcheck, v0.11.0 |
| uv | https://github.com/astral-sh/uv, 0.12.23 |
| govulncheck | https://go.dev/doc/security/vuln/, golang.org/x/vuln v1.8.0 |
| cargo-audit | https://github.com/rustsec/rustsec, cargo-audit 0.22.2 |
| pip-audit | https://github.com/pypa/pip-audit, 2.10.1 |
| osv-scanner | https://github.com/google/osv-scanner, v2.6.0 |
| checkov | https://github.com/bridgecrewio/checkov, 3.3.23 |
| NuGet's audit of packages | https://learn.microsoft.com/nuget/concepts/auditing-packages |
| npm audit and composer audit | https://docs.npmjs.com/cli/commands/npm-audit, https://getcomposer.org/doc/03-cli.md#audit |
| golangci-lint | https://golangci-lint.run, v2.14.0 |
| golangci-lint refuses a Go version newer than the one that built it | `pkg/goutil/version.go` of golangci-lint v2.14.0 |
| benchstat | https://pkg.go.dev/golang.org/x/perf/cmd/benchstat, golang.org/x/perf v0.0.0-20260929162123-406019bb8b68 |
| dokimi-mutate-go | https://github.com/dokimasia/mutate-go, go.dokimi.dev/mutate v0.0.0-20261006212535-719083ce3457 |
| ruff and mypy | https://docs.astral.sh/ruff/, 0.16.10, and https://mypy.readthedocs.io, 2.4.0 |
| clippy | https://doc.rust-lang.org/clippy/ |
| Biome | https://biomejs.dev, 2.5.15 |
| The analysis of .NET code | https://learn.microsoft.com/dotnet/fundamentals/code-analysis/overview |
| PMD and its base ruleset | https://pmd.github.io, 7.28.0 |
| ktlint | https://pinterest.github.io/ktlint/, 1.8.0 |
| PHPStan, its extension installer and PHP-CS-Fixer | https://phpstan.org, 2.2.17, https://github.com/phpstan/extension-installer, 1.4.3, and https://cs.symfony.com, 3.95.27 |
| tflint | https://github.com/terraform-linters/tflint, v0.64.0 |
| The cache of GitHub Actions, which saves after a job that succeeds | https://github.com/actions/cache, v6.1.0, `action.yml` |
| The delimiters of `text/template` | https://pkg.go.dev/text/template#Template.Delims |
| The analyzers of Go and their multichecker | https://pkg.go.dev/golang.org/x/tools/go/analysis/multichecker |
| Typed options of a tool in its language backend | Pants, `src/python/pants/backend/go/lint/golangci_lint/subsystem.py` |
| Managed files, files written once, and typed workflow options | projen, `FileBase`, `SampleFile`, `workflowRunsOn` and `workflowNodeVersion`, https://github.com/projen/projen |
