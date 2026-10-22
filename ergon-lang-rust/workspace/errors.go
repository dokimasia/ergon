// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import "errors"

// ErrManifest is the error for a Cargo.toml that a release cannot read: a file that does not parse,
// a value of the wrong type, a crate without a name, a version that does not parse, and a workspace
// whose members or inherited keys Cargo cannot resolve.
var ErrManifest = errors.New("workspace: invalid Cargo.toml")
