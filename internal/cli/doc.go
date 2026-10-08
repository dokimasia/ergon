// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package cli runs the command line of ergon, with cobra for the commands and viper for the
// configuration.
//
// [Run] builds the root command, runs it over the arguments under the caller's context, and
// returns the exit status. The help lists the languages of the catalog that the caller's
// registration fills.
//
// # Commands
//
//   - ergon init sets up the repository in the working directory with the baseline of ergon,
//     through the subcommands new, add, remove, check and sync of
//     [go.dokimi.dev/ergon/service/baseline]. The common files render first, the GitHub files
//     second, the license files third, and then the files of each language of the catalog.
//   - ergon init upgrade moves the repository to the baseline of the newest release of ergon. It
//     finds the release in the module proxy of Go with [go.dokimi.dev/ergon/service/pin], and
//     runs it through [go.dokimi.dev/ergon/service/tool] when it is newer than the running ergon.
//     The newest release syncs the repository and lists the pins of .ergon.yaml that differ from
//     its baseline. ergon init ci upgrade upgrades the repository in the same way in the workflow
//     baseline.yml, and proposes the change as a pull request through
//     [go.dokimi.dev/ergon/service/forge].
//   - ergon license check and ergon license fix check and fix the license headers of the files of
//     the repository with [go.dokimi.dev/ergon/service/licenses], by the section license of
//     .ergon.yaml.
//   - ergon release plans, writes and publishes the releases of the packages of the repository
//     with [go.dokimi.dev/ergon/service/release]. Its subcommands ci select-mode, ci version and ci
//     wait run the jobs of the release workflow through [go.dokimi.dev/ergon/service/forge].
//   - ergon tool run installs and runs a tool of a section of .ergon.yaml with
//     [go.dokimi.dev/ergon/service/tool], in the working directory, and exits with the exit status
//     of the tool.
//
// ergon license and ergon tool work on the repository of the working directory or of the nearest
// of its parents with .ergon/init.lock, so a target of the Makefile that runs in a directory of
// the repository, such as a module of Go, finds the options.
//
// # Configuration
//
// Before a command runs, the root command resolves the working directory with the Getwd function
// of the [Process], and reads the configuration file as YAML: the file that --config names, or
// .ergon.yaml in the working directory. A missing .ergon.yaml is no error. ergon init, ergon
// license and ergon tool read no configuration through viper, because they read the options of
// .ergon.yaml through the baseline of ergon init.
//
// # Errors
//
// Run writes every error to standard error after the name of the program. An error of the
// command line itself, such as an unknown command or flag, a missing required flag, or a section
// or a tool that ergon tool run does not find, ends with a pointer to the help and the exit status
// 2. Every other error has the exit status 1, among them a managed file that ergon init check finds
// edited or outdated and a file without the license header of the section license. A tool that
// ergon tool run runs sets the exit status itself.
//
// # Dependency position
//
// Imports the standard library, github.com/spf13/cobra, github.com/spf13/viper,
// [go.dokimi.dev/ergon/core/changeset], [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/spdx],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace],
// [go.dokimi.dev/ergon/service/baseline] with its producers common and github and its package
// lock, [go.dokimi.dev/ergon/service/forge], [go.dokimi.dev/ergon/service/licenses] with its
// producer of the license files, [go.dokimi.dev/ergon/service/pin],
// [go.dokimi.dev/ergon/service/release], [go.dokimi.dev/ergon/service/tool] and
// [go.dokimi.dev/ergon/service/vcs]. cmd/ergon imports it.
package cli
