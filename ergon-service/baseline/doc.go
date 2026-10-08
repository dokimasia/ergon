// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline sets up a repository with the files of ergon init and keeps them at the
// baseline of the installed ergon.
//
// A [Repository] renders the files of its producers: the base producers that [Open] receives,
// such as the common files, the GitHub files and the license files, then the producer of the
// toolchain and of each language of the answers. It compares the rendering with the repository and
// with the lock, .ergon/init.lock, and writes the files of the commands New, Add, Remove and Sync.
//
// Each command runs the same steps:
//
//  1. [go.dokimi.dev/ergon/service/baseline/options] resolves the options of each producer from
//     .ergon.yaml and the lock.
//  2. [go.dokimi.dev/ergon/service/baseline/render] collects the contributions of the producers to
//     the workflows, and renders their templates.
//  3. [go.dokimi.dev/ergon/service/baseline/overlay] merges each local file into its managed file,
//     and the producer of the managed file checks the result when it is a
//     [language.LocalChecker].
//  4. The command compares the rendering with the files and with
//     [go.dokimi.dev/ergon/service/baseline/lock], and writes or reports the difference.
//
// Every command but New reads the lock first, and fails for a lock that a newer release of ergon
// wrote than the running one. A build of ergon without a release, whose version is dev, orders
// against no release. Such a build writes only over a lock that such a build wrote, because a CI
// job installs the release that the lock names. New, Add, Remove and Sync fail for it otherwise.
//
// # Classes
//
//   - A managed file is the rendering, line for line. The lock records its digest, so
//     [Repository.Check] reports a file that is missing, edited by hand or outdated.
//   - .ergon.yaml receives the section of each producer that has options, with the answers in the
//     keys that state them, such as license.spdx. A command keeps every other key of the file, and
//     removes only the section of a producer that the repository no longer has. Every command but
//     New fails when such a key has neither the answer of the lock nor the answer that the command
//     sets.
//   - A seeded file is written when it is absent, and never again.
//
// # Writes
//
// Every command computes its whole change before it writes. It writes each file to a temporary
// file in the same directory and renames it, so an interrupted run leaves each file old or new.
// An [os.Root] confines every write and removal to the repository's directory.
//
// # Errors
//
// The sentinels classify the errors of the commands: [ErrInvalidOpen], [ErrInitialized],
// [ErrNotInitialized], [ErrNewerLock], [ErrDevelopmentBuild], [ErrUnknownLanguage],
// [ErrLanguagePresent], [ErrLanguageAbsent], [ErrUnsupported], [ErrConflict], [ErrInvalidFile],
// [ErrUnmanagedLocal] and [ErrUnknownSection], each of which starts with "baseline: ". A [ConflictError] wraps ErrConflict
// and lists the managed files that a command left. A command also returns the errors of the
// packages of its steps, such as [go.dokimi.dev/ergon/service/baseline/options.ErrInvalid] for
// .ergon.yaml that the producers do not accept or whose answers contradict the lock, and
// [language.ErrInvalidLocal] for a local file that a producer refuses.
//
// # Options
//
// [Repository.Options] resolves the options of one section as the commands resolve them, for a
// command that reads the options of a producer, such as ergon tool run or ergon license.
// [Repository.Overrides] returns the pins of .ergon.yaml whose version differs from the baseline,
// for the report of an upgrade of ergon.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace],
// [go.dokimi.dev/ergon/service/pin], and the packages lock, options, overlay and render of
// [go.dokimi.dev/ergon/service/baseline]. internal/cli of the root module imports it.
package baseline
