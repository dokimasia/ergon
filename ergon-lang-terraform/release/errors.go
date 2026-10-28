// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "errors"

// ErrRequirement is the error of [Versioner.Resolve] and [Versioner.Rewrite] for every requirement,
// because a package of Terraform has no requirement on another package of the repository.
var ErrRequirement = errors.New("release: a package of Terraform has no requirement")
