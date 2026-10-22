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

// ErrDistribution is the error of [Packer.Pack] for a build that does not write a distribution of
// the project at its version, and for a distribution of another project or version. It is also the
// error of [Publisher.Publish] for a pack directory without the distributions of a project at its
// version.
var ErrDistribution = errors.New("release: no distribution of the release")

// ErrIndex is the error of [Publisher.Published] for a status of the JSON API of PyPI other than
// 200 and 404.
var ErrIndex = errors.New("release: unexpected status of the index")
