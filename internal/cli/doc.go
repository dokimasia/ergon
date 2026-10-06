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
// ergon init sets up the repository in the working directory with the baseline of ergon, through
// the subcommands new, add, remove, check and sync of [go.dokimi.dev/ergon/service/baseline]. The
// common files render first, the GitHub files second, and then the files of each language of the
// catalog.
//
// # Configuration
//
// Before a command runs, the root command resolves the working directory with the Getwd function
// of the [Process], and reads the configuration file as YAML: the file that --config names, or
// .ergon.yaml in the working directory. A missing .ergon.yaml is no error. ergon init reads no
// configuration, because it writes .ergon.yaml.
//
// # Errors
//
// Run writes every error to standard error after the name of the program. An error of the
// command line itself, such as an unknown command or flag or a missing required flag, ends with a
// pointer to the help and the exit status 2. Every other error has the exit status 1, among them
// a managed file that ergon init check finds edited or outdated.
//
// # Dependency position
//
// Imports the standard library, github.com/spf13/cobra, github.com/spf13/viper,
// [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/workspace],
// [go.dokimi.dev/ergon/service/baseline], [go.dokimi.dev/ergon/service/baseline/common] and
// [go.dokimi.dev/ergon/service/baseline/github]. cmd/ergon imports it.
package cli
