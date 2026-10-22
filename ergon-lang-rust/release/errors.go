// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "errors"

// ErrRequirement is the error for a text that is not a version requirement of Cargo, and for a
// requirement that a release cannot move to admit a new version with its operators.
var ErrRequirement = errors.New("release: invalid version requirement")

// ErrEdit is the error of [Versioner.Apply] for an edit whose crate has a manifest without the
// version or the requirements that its discovery found, and for two edits that write two
// requirements in place of one inherited requirement.
var ErrEdit = errors.New("release: edit of a changed manifest")

// ErrCrate is the error of [Packer.Pack] for a crate whose .crate file cargo package does not write.
// It is also the error of [Publisher.Publish] for a pack directory without the .crate file of a
// crate.
var ErrCrate = errors.New("release: no package of the crate")
