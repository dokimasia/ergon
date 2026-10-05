// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo

// The variables that a build sets with -X flags of the linker.
var (
	// version is the version of the release, such as 1.2.3.
	version = ""

	// commit is the commit of the release.
	commit = ""

	// date is the date of the commit.
	date = ""
)

// development is the version of a build without the flags of the linker.
const development = "dev"

// Full returns the version of this build, as [Format] writes the variables that the build sets.
func Full() string {
	return Format(version, commit, date)
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
