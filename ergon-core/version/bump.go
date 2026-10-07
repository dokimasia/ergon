// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package version

import (
	"fmt"
	"slices"
)

// Bump is the level of a release that a changeset names for a package, as changesets spells it in
// the front matter of a changeset file. The zero value is not a valid level.
type Bump string

// The levels of a release, from the lowest to the highest.
const (
	// BumpNone releases nothing. A changeset names it to state that a change of a package needs no
	// release.
	BumpNone Bump = "none"

	// BumpPatch releases a fix: 1.2.3 becomes 1.2.4.
	BumpPatch Bump = "patch"

	// BumpMinor releases a feature: 1.2.3 becomes 1.3.0.
	BumpMinor Bump = "minor"

	// BumpMajor releases a breaking change: 1.2.3 becomes 2.0.0, and 0.4.2 becomes 1.0.0.
	BumpMajor Bump = "major"
)

// bumps are the levels, from the lowest to the highest.
var bumps = []Bump{BumpNone, BumpPatch, BumpMinor, BumpMajor}

// ParseBump returns the level that s spells. It returns an error that wraps [ErrInvalid] for an s
// that is none of none, patch, minor and major.
func ParseBump(s string) (Bump, error) {
	b := Bump(s)
	if !slices.Contains(bumps, b) {
		return "", fmt.Errorf("%w: level %q, which is none of none, patch, minor and major", ErrInvalid, s)
	}
	return b, nil
}

// Valid reports whether b is one of the four levels.
func (b Bump) Valid() bool {
	return slices.Contains(bumps, b)
}

// Max returns the higher of b and c. A level that is not valid is lower than every valid level,
// so Max returns the valid one of the two.
func (b Bump) Max(c Bump) Bump {
	if slices.Index(bumps, c) > slices.Index(bumps, b) {
		return c
	}
	return b
}

// AtLeast reports whether b is c or a higher level. A level that is not valid is at least no
// valid level.
func (b Bump) AtLeast(c Bump) bool {
	return b.Valid() && slices.Index(bumps, b) >= slices.Index(bumps, c)
}
