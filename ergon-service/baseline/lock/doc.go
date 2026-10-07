// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package lock reads and writes .ergon/init.lock, the lock of ergon init.
//
// A [Lock] records the release of ergon that wrote it, the digest of each managed file, the
// baseline value of each option of .ergon.yaml, and the answers. A repository commits it with its
// files, so ergon init check compares the files with the lock in every clone.
//
// [Decode] parses the lock strictly: an unknown field, data after the object, and an entry of a
// file that breaks the rules of [File] return an error that wraps [ErrInvalid]. [Lock.Encode]
// writes JSON indented by two spaces with a final newline, so the same lock always encodes to the
// same bytes.
//
// # Dependency position
//
// Imports the standard library and [go.dokimi.dev/ergon/core/language]. The package
// [go.dokimi.dev/ergon/service/baseline] and the tool runner import it.
package lock
