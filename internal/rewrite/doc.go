// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package rewrite writes resolved releases into the baselines of the producers of ergon init.
//
// A producer states its baseline in the composite literal that its Options method returns. [Apply]
// follows the field names of each pin of [go.dokimi.dev/ergon/service/pin] from that literal to the
// string literals of the pin, and replaces them with the values of the release that an [Update]
// names:
//
//   - the version of a tool, and of a version with a source tag
//   - the commit and the release of an action
//   - the version and the digest of each platform of a release binary
//
// The files keep their comments and their layout, and Apply formats each file that it changes with
// gofmt. A pin whose value a function or a constant computes has no literal to rewrite, so the
// value of a pin is a string literal or a composite literal in the method, or the value of a
// package-level variable.
//
// # Errors
//
// Apply returns an error that wraps [ErrNoOptions] for a package without the Options method of the
// producer, [ErrNotLiteral] for a pin whose value is no literal, and [ErrMismatch] for a literal
// whose value differs from the pin. The error of a pin names its key.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/option],
// [go.dokimi.dev/ergon/core/workflow] and [go.dokimi.dev/ergon/service/pin]. The command
// update-baseline of ergon's repository imports it.
package rewrite
