// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Tagger is the [language.Tagger] of the toolchain terraform. The zero value is ready for use, and
// it is safe for concurrent use.
type Tagger struct{}

var _ language.Tagger = Tagger{}

// Tag returns v and the version, such as v1.2.0, also in a repository with packages of other
// toolchains, because the package of Terraform is the whole repository.
func (Tagger) Tag(_ *workspace.Package, v version.Version) string {
	return "v" + v.String()
}
