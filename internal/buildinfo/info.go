// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo

import (
	"runtime/debug"
	"strings"
	"time"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// Development is the version of a build that no release tags.
const Development = "dev"

// The keys of the build settings that the go command records for a build of a git checkout.
const (
	revisionKey = "vcs.revision"
	timeKey     = "vcs.time"
	modifiedKey = "vcs.modified"
)

// The parts of the version string of [Info.String].
const (
	// shortRevision is the number of hexadecimal digits of the commit in the version string, the
	// abbreviation of git.
	shortRevision = 7

	// dateLayout is the layout of the date of the commit in the version string.
	dateLayout = "2006-01-02"

	// modifiedNote marks a build of a working tree with changes in the version string.
	modifiedNote = "modified"
)

// dirty are the suffixes that the go command appends to the version of a build of a working tree
// with changes: +dirty, and .dirty after the build metadata +incompatible.
var dirty = []string{"+dirty", ".dirty"}

// Info is the build information of a binary. The zero value is no valid Info, because its Version
// is empty.
type Info struct {
	// Time is the time of the commit in UTC, and the zero time for a build without it.
	Time time.Time

	// Version is the version of the release without its v, such as 0.6.0, or [Development].
	Version string

	// Revision is the commit of the build, or empty for a build without it, such as go install of
	// a version.
	Revision string

	// Modified reports that the working tree of the build had changes.
	Modified bool
}

// Read returns the Info of the running binary, as [From] reads the build information of
// [debug.ReadBuildInfo].
func Read() Info {
	info, _ := debug.ReadBuildInfo()
	return From(info)
}

// From returns the Info of info. The version is the version of the main module without its v when
// a release tags the commit, as go build at a tag and go install of a version record it. It is
// [Development] for a nil info, for a version that is no semantic version, such as (devel), for a
// pseudo-version, and for a build of a working tree with changes. From reads the settings
// vcs.revision, vcs.time and vcs.modified, and leaves Time zero for a time that is not RFC 3339.
func From(info *debug.BuildInfo) Info {
	if info == nil {
		return Info{Version: Development}
	}
	var i Info
	for _, s := range info.Settings {
		switch s.Key {
		case revisionKey:
			i.Revision = s.Value
		case timeKey:
			if t, err := time.Parse(time.RFC3339Nano, s.Value); err == nil {
				i.Time = t.UTC()
			}
		case modifiedKey:
			i.Modified = s.Value == "true"
		}
	}
	v := info.Main.Version
	for _, suffix := range dirty {
		if strings.HasSuffix(v, suffix) {
			i.Modified = true
		}
	}
	i.Version = Development
	if semver.IsValid(v) && !module.IsPseudoVersion(v) && !i.Modified {
		i.Version = strings.TrimPrefix(v, "v")
	}
	return i
}

// String returns the version string of i: the version, followed by the short commit, the date of
// the commit and the note modified in parentheses, each where i has it, such as 0.6.0 (a5eb82a,
// 2026-10-08) or dev (a5eb82a, 2026-10-08, modified). It returns the version alone for an i without
// a commit, a time and changes.
func (i Info) String() string {
	parts := make([]string, 0, 3)
	if i.Revision != "" {
		parts = append(parts, i.Revision[:min(shortRevision, len(i.Revision))])
	}
	if !i.Time.IsZero() {
		parts = append(parts, i.Time.Format(dateLayout))
	}
	if i.Modified {
		parts = append(parts, modifiedNote)
	}
	if len(parts) == 0 {
		return i.Version
	}
	return i.Version + " (" + strings.Join(parts, ", ") + ")"
}
