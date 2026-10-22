// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "errors"

// ErrEdit is the error of [Versioner.Apply] for an edit whose package has a manifest that lacks the
// version or the requirements that its discovery found.
var ErrEdit = errors.New("release: edit of a changed manifest")

// ErrTarball is the error of [Packer.Pack] for a pack that does not write the tarball of the
// package at its version. It is also the error of [Publisher.Publish] for a pack directory without
// the tarball of a package at its version.
var ErrTarball = errors.New("release: no tarball of the release")

// ErrRegistry is the error of [Publisher.Published] for a status of the npm registry other than 200
// and 404.
var ErrRegistry = errors.New("release: unexpected status of the registry")
