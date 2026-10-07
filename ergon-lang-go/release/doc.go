// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release implements the release roles of the toolchain go: the [Versioner] of require
// lines, the [Tagger] of the tags that the go command reads, and the [Locker] of the hashes that
// go.sum records.
//
// # Requirements
//
// A require line names the lowest version of a module that it admits, and minimal version
// selection gives a consumer that version until another go.mod requires a higher one. A require
// line therefore pins every newer version of the module that it names, and a release of a module
// reaches the consumers of a module that requires it only through a release of that module.
//
// # One commit
//
// [Versioner.Apply] writes the require lines of a release into go.mod, and refreshes the go.sum of
// each module that requires a released module without a directory replace. go.sum needs the hash
// of the released version before its tag exists, so Apply serves the module from a module proxy
// in a temporary directory, which writes the zip of the module from a snapshot of the working
// tree with golang.org/x/mod/zip. The hash of that zip is the hash of the module at its tag, because
// the zip and the go command read the files of a module with the same git archive. Apply then runs
// go mod tidy in each such module, a module after every module that it requires.
//
// # Pending releases
//
// A change to a released module between the version commit and its tag changes the content that
// the tag names, while the go.sum of a module that requires it still records the content of the
// version commit. [Locker.Stale] reports such a go.sum. [Locker.Lock] removes its lines of the
// released modules and runs go mod tidy against the proxy of [Versioner.Apply], so that the go.sum
// records the content that the tag names.
//
// # Workspaces
//
// The go command reads the go.mod of every version of a module of the workspace that another
// module of the workspace requires, also before the tag of the version exists, such as between the
// merge of a version pull request and its tags. Apply therefore writes a replace of each such
// version by the directory of its module into go.work, and removes the replace of a version that no
// go.mod requires. A replace in go.work applies in the workspace alone, so it changes no build of a
// consumer and no go install.
//
// # Errors
//
// Each error of the package wraps [ErrRequirement], [ErrMajor] or [ErrCycle], or the error of the
// go command, of git or of the file system, and names the module or the file that caused it.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/mod/modfile, golang.org/x/mod/module,
// golang.org/x/mod/sumdb/dirhash, golang.org/x/mod/zip, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace] and
// [go.dokimi.dev/ergon/lang/go/workspace].
package release
