// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "errors"

// ErrSpecifier is the error for a requirement whose specifier is not a version specifier of
// PEP 440, and for a specifier that a release cannot move to a new version with its operators.
var ErrSpecifier = errors.New("release: invalid version specifier")

// ErrEdit is the error of [Versioner.Apply] for an edit whose package has a manifest without the
// version or the requirements that its discovery found.
var ErrEdit = errors.New("release: edit of a changed manifest")
