// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import "errors"

// ErrVersion is the error for a text that is not a version of PEP 440, and for a version that the
// other scheme of a conversion does not have, such as a post-release in Semantic Versioning.
var ErrVersion = errors.New("workspace: invalid version")

// ErrRequirement is the error for a text that is not a requirement of PEP 508.
var ErrRequirement = errors.New("workspace: invalid requirement")

// ErrManifest is the error for a pyproject.toml that a release cannot read: a file that does not
// parse, a value of the wrong type, a project without a name or a version, and a requirement or a
// version that does not parse.
var ErrManifest = errors.New("workspace: invalid pyproject.toml")
