// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo

import (
	"runtime/debug"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// The variables that a build sets with -X flags of the linker.
var (
	// version is the version of the release, such as 1.2.3.
	version = ""

	// commit is the commit of the release.
	commit = ""

	// date is the date of the commit.
	date = ""
)

// development is the version of a build without a release.
const development = "dev"

// Version returns the version of the release of this build, such as 1.2.3, as [Release] resolves
// it from the build information of the binary, and dev for a build without a release.
func Version() string {
	info, _ := debug.ReadBuildInfo()
	return Format(Release(version, info), "", "")
}

// Full returns the version of this build, as [Format] writes the release that [Release] resolves
// and the commit and the date that the build sets.
func Full() string {
	info, _ := debug.ReadBuildInfo()
	return Format(Release(version, info), commit, date)
}

// Release returns the version of the release of a build: linked, the version that the flags of the
// linker set, and else the version of the main module of info, without its v, when a release tags
// it, as go install go.dokimi.dev/ergon/cmd/ergon@v1.2.3 records v1.2.3. It returns the empty
// string for a build without either: a nil info, the version (devel) that the go command records
// for a build of a working tree, and a pseudo-version, which it records for go install at a commit.
func Release(linked string, info *debug.BuildInfo) string {
	if linked != "" {
		return linked
	}
	if info == nil || !semver.IsValid(info.Main.Version) || module.IsPseudoVersion(info.Main.Version) {
		return ""
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}

// Format returns the version string of a build of version, commit and date:
//
//   - dev for an empty version
//   - the version for an empty commit
//   - the version with the commit in parentheses for an empty date
//   - the version with the commit and the date in parentheses otherwise
//
// Format("1.2.3", "abc123", "2026-01-01") returns "1.2.3 (abc123, built 2026-01-01)".
func Format(version, commit, date string) string {
	if version == "" {
		return development
	}
	if commit == "" {
		return version
	}
	if date == "" {
		return version + " (" + commit + ")"
	}
	return version + " (" + commit + ", built " + date + ")"
}
