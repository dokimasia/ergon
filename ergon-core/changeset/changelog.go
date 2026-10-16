// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package changeset

import (
	"regexp"

	"go.dokimi.dev/ergon/core/version"
)

// versionHeading matches a heading of a changelog with a version, such as ## 1.2.0, and captures
// the version in its first group.
var versionHeading = regexp.MustCompile(`(?m)^#{1,6}[ \t]+v?(\d+\.\d+\.\d+\S*)[ \t]*$`)

// Latest returns the highest version of the headings of the changelog data, such as 1.2.0 of
// ## 1.2.0, and the zero version for a changelog without such a heading. A heading may write a v
// before the version. A heading counts only when its text is a version of Semantic Versioning
// 2.0.0.
func Latest(data []byte) version.Version {
	var highest version.Version
	for _, m := range versionHeading.FindAllSubmatch(data, -1) {
		if v, err := version.Parse(string(m[1])); err == nil && v.Compare(highest) > 0 {
			highest = v
		}
	}
	return highest
}
