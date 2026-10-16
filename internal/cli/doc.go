// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package cli runs the command line of ergon on go.stealthscale.io/cli.
//
// [Program] is what ergon needs of the process that runs it, the registration of its languages and
// its version. [Program.Main] runs the command line of a [gocli.Env] under the caller's context and
// returns the exit status, as the method Main of a program of go-cli does. The help lists the
// languages of the catalog that the registration fills.
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
//     with [go.dokimi.dev/ergon/service/release]. Its subcommands ci select-mode, ci verify and ci
//     version run the jobs of the release workflows, and ci skip the first job of ci.yml, through
//     [go.dokimi.dev/ergon/service/forge].
//   - ergon tool run installs and runs a tool of a section of .ergon.yaml with
//     [go.dokimi.dev/ergon/service/tool], in the working directory, and exits with the exit status
//     of the tool.
//   - ergon completion writes the completion script of a shell. go-cli adds the command.
//
// ergon license and ergon tool work on the repository of the working directory or of the nearest
// of its parents with .ergon/init.lock, so a target of the Makefile that runs in a directory of
// the repository, such as a module of Go, finds the options.
//
// # Options
//
// The options of each command are the fields of a struct, from which go-cli derives the flag and
// the variable of the environment of each option, such as --owner and ERGON_INIT_NEW_OWNER. A flag
// replaces the value of the variable. The commands read the options of .ergon.yaml through the
// baseline of ergon init, and no option of the command line comes from a file. The hook of the app
// resolves the working directory with the Getwd function of the [Program] before a command runs.
//
// # Errors
//
// Main writes every error to standard error after the name of the program. An error of the command
// line itself, such as an unknown command or flag, a missing required flag, or a section or a tool
// that ergon tool run does not find, has the exit status 2. A command line that ends at a command
// without its subcommand, such as ergon init, writes the help of that command first. Every other
// error has the exit status 1, among them a managed file that ergon init check finds edited or
// outdated and a file without the license header of the section license. A tool that ergon tool run
// runs sets the exit status itself.
//
// # Dependency position
//
// Imports the standard library, go.stealthscale.io/cli, [go.dokimi.dev/ergon/core/changeset],
// [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/option],
// [go.dokimi.dev/ergon/core/spdx], [go.dokimi.dev/ergon/core/version],
// [go.dokimi.dev/ergon/core/workspace], [go.dokimi.dev/ergon/service/baseline] with its producers
// common and github and its package lock, [go.dokimi.dev/ergon/service/forge],
// [go.dokimi.dev/ergon/service/licenses] with its producer of the license files,
// [go.dokimi.dev/ergon/service/pin], [go.dokimi.dev/ergon/service/release],
// [go.dokimi.dev/ergon/service/tool] and [go.dokimi.dev/ergon/service/vcs]. cmd/ergon imports it.
package cli
