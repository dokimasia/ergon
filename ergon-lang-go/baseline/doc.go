// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Go of ergon init: the configuration of golangci-lint, the
// configuration of GoReleaser of each module with a command, the fragments of Go of the shared
// files, the section go of .ergon.yaml, and the part of Go of the workflows.
//
// [Producer] renders .golangci.yml as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/. .golangci.yml
// enables the linter of each module plugin of the key lint.plugins, which ergon tool run builds
// into golangci-lint.
//
// # Makefile
//
// The fragment of the Makefile runs each target in every module of go.work: fmt-go, lint-go,
// test-go, race-go, fuzz-go, bench-go, benchstat-go, mutate-go, generate-go, verify-generate-go and
// audit-go, and check-go, which requires the targets of the steps that the key check of the section
// names. Every tool runs through ergon tool run, which installs the version that the section names.
// The fragment states each option of a step as a variable, such as GO_FUZZ_TIME, which one run of
// make overrides on its command line. lint-go also runs ergon-go-vet. generate-go runs the command
// of the generators, go generate at the baseline, and verify-generate-go fails when that command
// changes a file of the repository. A line ##@ Go starts the group of Go in make help.
//
// # Binaries
//
// The key binaries of the section lists each [Command] whose binaries each release of its module
// attaches. As a placer, [Producer.Files] renders the configuration of GoReleaser of each module
// that a command names, .goreleaser.yaml in the directory of the module, from goreleaser/. The
// configuration builds each command with CGO_ENABLED=0, -trimpath and -ldflags=-s -w, without -X
// flags, because Go writes the version of the module from its tag, and Go applies a default.pgo
// beside the main package. A post hook of the build <name>-upx packs each Linux binary of a command
// with UPX and LZMA, whose packed binaries syft still reads. The build <name> of the macOS and
// Windows targets has no hook, because UPX refuses macOS binaries and syft reads no packed Windows
// binary. The configuration writes:
//
//   - an archive .tar.gz of each command and platform, with LICENSE, README.md and the completions
//     of bash, zsh and fish of a cobra program
//   - the deb, rpm and apk packages of a command, with its completions and its license
//   - a source archive, checksums.txt, its cosign signature without a key, and an SPDX SBOM of each
//     archive and package
//   - a cask of Homebrew of each command with a cask, for the tap of the key homebrew
//
// GoReleaser runs in snapshot mode with the version of the release in ERGON_VERSION, and creates no
// release and no changelog: ergon release pack runs it, and ergon release publish attaches the
// assets. [GoReleaser], [Cosign], [Syft] and [UPX] are the release binaries of the section, pinned
// by version and a SHA-256 per platform. The fragment of .gitignore ignores /dist/, so a build at the
// tag of a release sees no change of the working tree.
//
// # Options
//
// [Options] is the section go: the tools, the package patterns, the steps of the gate, the steps of
// nightly.yml with the limit of each, the options of each step, the commands and the tap of their
// casks, and the key ci of the job check-go. [Options.Contribution] returns the job check-go, a job
// of nightly.yml for each step of the key nightly, the setup of Go in the jobs of release.yml, the
// assets of the releases of the commands, the CodeQL analysis of go and the updates of the modules.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/spdx] and
// [go.dokimi.dev/ergon/core/workflow]. The root package of the module imports it.
package baseline
