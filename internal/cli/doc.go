// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package cli runs the command line of ergon, with cobra for the commands and viper for the
// configuration.
//
// [Run] builds the root command, runs it over the arguments under the caller's context, and
// returns the exit status. The help lists the languages of the catalog that the caller's
// registration fills.
//
// # Configuration
//
// Before any command runs, the root command reads the configuration file as YAML: the file that
// --config names, or .ergon.yaml in the working directory. A missing .ergon.yaml is no error.
//
// # Errors
//
// Run writes every error to standard error after the name of the program. An error of the
// command line itself, such as an unknown command or flag, ends with a pointer to the help and
// the exit status 2. Every other error has the exit status 1.
//
// # Dependency position
//
// Imports the standard library, github.com/spf13/cobra, github.com/spf13/viper and
// [go.dokimi.dev/ergon/core/language]. cmd/ergon imports it.
package cli
