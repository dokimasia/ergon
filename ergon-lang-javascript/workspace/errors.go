// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import "errors"

// ErrManifest is the error for a package.json or a pnpm-workspace.yaml that a release cannot read,
// and for a package that lacks the name or the version that a release needs. [ParseManifest] and
// [Discover] return it.
var ErrManifest = errors.New("workspace: invalid manifest")
