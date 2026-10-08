// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package vcs runs git for the commands of ergon.
//
// # Files
//
// [Files] lists the files of a working tree that git tracks or that it would track, as git
// ls-files lists them, so a command works on the files of the repository and skips every file that
// a .gitignore excludes. [Changed] lists the files that a branch and its working tree change since
// the merge base of a base, such as origin/main, which ergon release status reads.
//
// # History and tags
//
// [Head] returns the commit of HEAD, and [AddedBy] the commit that added a file, which names the
// pull request of a changeset. [Tags] returns the commit of each tag, [Tag] creates an annotated
// tag, which git signs when tag.gpgSign is set, and [Push] pushes refs in one atomic push. Both run
// git with a [Terminal], on which a signing program and ssh ask for the PIN and the touch of a key.
//
// # Snapshots
//
// [Snapshot] writes the tree of the working tree, tracked and untracked files alike, through an
// index of its own, and [Restore] gives paths their content in such a tree again. ergon release
// version takes a snapshot before it writes, and restores the paths that it wrote when a step
// fails.
//
// git runs with the environment of the process, so a command that a hook of git runs reads the
// repository of the hook.
//
// # Errors
//
// A command that git cannot run, or that git ends with an error, returns an error that wraps
// [ErrGit] and states the arguments, and the standard error of git unless git wrote it to the
// Stderr of a [Terminal].
//
// # Dependency position
//
// Imports the standard library. The packages [go.dokimi.dev/ergon/service/licenses] and
// [go.dokimi.dev/ergon/service/release] import it, and the composition root of ergon passes its
// functions to the toolchain of Go.
package vcs
