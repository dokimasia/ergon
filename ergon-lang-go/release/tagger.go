// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Tagger is the [language.Tagger] of the toolchain go. It names the tags that the go command reads
// the version of a module from. The zero value is ready for use.
type Tagger struct{}

var _ language.Tagger = Tagger{}

// Tag returns the tag of the module p at v: v<version> for the module at the root of the
// repository, and <dir>/v<version> for a module in the directory <dir>.
func (Tagger) Tag(p *workspace.Package, v version.Version) string {
	return tagName(p.Dir, "v"+v.String())
}

// tagName returns the tag of the module in the directory dir at the version v of the go command,
// such as v1.2.0.
func tagName(dir, v string) string {
	if dir == "." {
		return v
	}
	return dir + "/" + v
}
