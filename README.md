<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ergon

[![CI](https://github.com/dokimasia/ergon/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/dokimasia/ergon/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/dokimasia/ergon)](https://github.com/dokimasia/ergon/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/go.dokimi.dev/ergon.svg)](https://pkg.go.dev/go.dokimi.dev/ergon)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

ergon sets up a repository with a managed baseline. It also keeps the license header of every file, and releases the packages of the repository. It works on repositories in one or more of eleven languages: C#, Java, Kotlin, PHP, JavaScript, TypeScript, Go, Python, Rust, Terraform and Bash.

| Command | What it does |
|---|---|
| `ergon init` | Renders the baseline: the GitHub workflows, the Makefile, the hooks of pre-commit, one linter configuration per language, and the license files |
| `ergon license` | Adds and checks the license header of every file |
| `ergon release` | Plans the releases from changeset files, writes the versions and the changelogs in a version pull request, and publishes the packages when the pull request merges |
| `ergon tool run` | Installs and runs the tools of `.ergon.yaml` at their pinned versions |

## Install

Each [release](https://github.com/dokimasia/ergon/releases/latest) has archives for Linux, macOS and Windows on amd64 and arm64, and `checksums.txt` with the SHA-256 digest of each archive. Download the archive for your system, check its digest, and put `ergon` on your `PATH`.

With Go 1.27.1 or later, install the command from source:

```sh
go install go.dokimi.dev/ergon/cmd/ergon@latest
```

`ergon --version` prints the version of the release.

## Quick start

Set up a repository with Go:

```sh
ergon init new --language go --owner "Example B.V." --license MIT \
  --repository example/demo --security-contact security@example.com
make help
make check
```

`ergon init new` writes the managed files, the seeded files such as `README.md`, `.ergon.yaml` and `.ergon/init.lock`. Commit the lock with the files. Then:

- Add a language with `ergon init add typescript`.
- Change an option in `.ergon.yaml`, such as the version of a tool, and run `ergon init sync`.
- Run `ergon init check` to list the managed files that differ from the baseline.
- Run `ergon init upgrade` to move to the baseline of the newest release of ergon.

## The baseline

ergon init sorts each file of the baseline into a class:

| Class | Examples | Who changes it |
|---|---|---|
| Managed | `Makefile`, `.github/workflows/ci.yml`, `.golangci.yml` | ergon init renders it. Write a setting of the repository to the same path under `.ergon/local`, and ergon init merges it in |
| Seeded | `README.md` | ergon init writes it when it is absent, and the repository maintains it |
| Options | `.ergon.yaml` | The repository changes an option, and ergon init renders the managed files from it |
| Lock | `.ergon/init.lock` | ergon init records its answers, the baseline value of each option and the digest of each managed file |

The Makefile has the same targets in every repository:

| Target | Runs |
|---|---|
| `make fmt` | The formatters of every language |
| `make lint` | The linters of every language |
| `make test` | The tests of every language |
| `make generate` | The generators of every language |
| `make audit` | The vulnerability scan of every language |
| `make check` | The gate of every language, as CI runs it |
| `make help` | The targets in groups: the common targets, each language, and the targets of the repository |

Each language adds its own targets, such as `lint-go` and `check-go`. The hooks of pre-commit run `make lint` and `make test` before each commit, and `make check` before each push. The option `common.hooks` of `.ergon.yaml` changes the targets of each stage.

## Releases

ergon release follows the workflow of changesets:

1. A pull request adds a changeset with `ergon release add`. The changeset lists each package that the change releases, with its bump.
2. When the CI run of a push to `main` passes, `version.yml` opens or updates the version pull request. It writes the new versions, the requirements between the packages and the changelogs.
3. The merge of the version pull request publishes the packages. `release.yml` checks that a CI run passed on the content of the commit, and then packs, tags and publishes each package.

`ergon release status` reports each package that a branch changed without a changeset. A run of `ci.yml` skips its jobs when a passed run already covers its content, such as the run of a version pull request whose parent passed. A repository can set the variable `ERGON_APP_CLIENT_ID` and the secret `ERGON_APP_PRIVATE_KEY` of a GitHub App, so that the checks of the version pull request start without an approval.

ergon release publishes Go modules. [The roadmap](docs/roadmap/README.md) lists the releases of the other toolchains.

## Documentation

| Directory | Contents |
|---|---|
| [docs/architecture](docs/architecture) | The modules and the packages, and what each may import |
| [docs/rfc](docs/rfc) | The designs and the argument behind them |
| [docs/adr](docs/adr) | The decisions and what they cost |
| [docs/roadmap](docs/roadmap) | The order of the work |

## Development

The targets of the Makefile run their tools through `ergon tool run`, so install ergon from the checkout first:

```sh
go install ./cmd/ergon
make check
```

`make check` runs the gate of the repository: the linters, the tests, the tests under the race detector and the vulnerability scan of every module. [CONTRIBUTING.md](CONTRIBUTING.md) describes how to propose a change, and [SECURITY.md](SECURITY.md) how to report a vulnerability.

## License

ergon is licensed under the MIT license, copyright Dokimasia B.V. See [LICENSE](LICENSE).
