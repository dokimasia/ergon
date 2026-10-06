// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline sets up a repository with the files of ergon init and keeps them at the
// baseline of the installed ergon.
//
// A [Repository] renders the files of its producers: the base producers that [Open] receives,
// such as the common files and the GitHub files, and the [language.Initializer] of each language
// of the answers. It joins the fragments of a shared file in the order of the producers, merges
// each local file into its managed file, and compares the result with the repository and with
// the lock, .ergon/init.lock.
//
// # Classes
//
//   - A managed file is the rendering, line for line. The lock records its digest, so
//     [Repository.Check] reports a file that is missing, edited by hand or outdated.
//   - A configured file, such as .ergon.yaml, receives the keys of its rendering. A command leaves
//     every other key as the repository wrote it.
//   - A seeded file is written when it is absent, and never again.
//
// # Local files
//
// A repository adds its own settings to a managed file through .ergon/local/<path>. A YAML file
// is merged into the rendering: maps key by key, with the local value for a key that both have,
// and lists appended. Any other file is appended to the rendering. A local file for a path that
// is not managed is an error.
//
// # Writes
//
// Every command computes its whole change before it writes. It writes each file to a temporary
// file in the same directory and renames it, so an interrupted run leaves each file old or new.
// An [os.Root] confines every write and removal to the repository's directory.
//
// # Errors
//
// The sentinels classify the errors: [ErrInitialized], [ErrNotInitialized], [ErrUnknownLanguage],
// [ErrLanguagePresent], [ErrLanguageAbsent], [ErrUnsupported], [ErrConflict], [ErrInvalidFile],
// [ErrUnmanagedLocal] and [ErrInvalidLock]. Every error starts with "baseline: ".
//
// # Dependency position
//
// Imports the standard library, go.yaml.in/yaml/v3, [go.dokimi.dev/ergon/core/language] and
// [go.dokimi.dev/ergon/core/workspace]. internal/cli of the root module imports it.
package baseline
