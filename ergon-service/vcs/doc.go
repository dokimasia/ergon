// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package vcs runs git for the commands of ergon.
//
// [Files] lists the files of a working tree that git tracks or that it would track, as git
// ls-files lists them, so a command works on the files of the repository and skips every file that
// a .gitignore excludes.
//
// git runs with the environment of the process, so a command that a hook of git runs reads the
// repository of the hook.
//
// # Errors
//
// A command that git cannot run, or that git ends with an error, returns an error that wraps
// [ErrGit] and states the arguments and the standard error of git.
//
// # Dependency position
//
// Imports the standard library. The package [go.dokimi.dev/ergon/service/license] imports it.
package vcs
