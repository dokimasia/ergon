// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package version

import (
	"cmp"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MaxComponent is the highest value of the major, the minor and the patch component of a
// [Version]: 2^53 - 1, the bound of node-semver.
const MaxComponent = 1<<53 - 1

// ErrInvalid is the error of [Parse] for a string that is no version of Semantic Versioning 2.0.0,
// and of [ParseBump] for a string that is no level.
var ErrInvalid = errors.New("version: invalid version")

// ErrOverflow is the error of [Version.Bump] for a component that the bump would move past
// [MaxComponent].
var ErrOverflow = errors.New("version: component past its bound")

// Version is a version of Semantic Versioning 2.0.0. The zero value is 0.0.0, the version of a
// package that has never been released.
type Version struct {
	// Pre is the pre-release without its hyphen, such as rc.1, or empty for a release.
	Pre string

	// Build is the build metadata without its plus sign, such as 20261007, or empty. It plays no
	// part in the order of versions.
	Build string

	// Major, Minor and Patch are the numeric components, each at most [MaxComponent].
	Major, Minor, Patch uint64
}

// Parse returns the version that s states, such as 1.4.0, 2.0.0-rc.1 or 1.0.0+20261007. s has no
// leading v.
//
// It returns an error that wraps [ErrInvalid] for an s that does not have three numeric
// components, for a component with a leading zero or above [MaxComponent], and for a pre-release
// or build identifier that is empty or has a character other than an ASCII letter, a digit and a
// hyphen, or a numeric pre-release identifier with a leading zero.
func Parse(s string) (Version, error) {
	rest, build, hasBuild := strings.Cut(s, "+")
	core, pre, hasPre := strings.Cut(rest, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("%w: %q, which does not have the form MAJOR.MINOR.PATCH", ErrInvalid, s)
	}
	var numbers [3]uint64
	for i, p := range parts {
		n, err := component(p)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %q: %w", ErrInvalid, s, err)
		}
		numbers[i] = n
	}
	v := Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}
	if hasPre {
		if err := identifiers(pre, true); err != nil {
			return Version{}, fmt.Errorf("%w: %q: the pre-release %w", ErrInvalid, s, err)
		}
		v.Pre = pre
	}
	if hasBuild {
		if err := identifiers(build, false); err != nil {
			return Version{}, fmt.Errorf("%w: %q: the build metadata %w", ErrInvalid, s, err)
		}
		v.Build = build
	}
	return v, nil
}

// String returns v as Parse reads it, such as 1.4.0 or 2.0.0-rc.1+20261007.
func (v Version) String() string {
	s := strconv.FormatUint(v.Major, 10) + "." + strconv.FormatUint(v.Minor, 10) + "." +
		strconv.FormatUint(v.Patch, 10)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// MarshalText returns v as [Version.String] writes it, so encoding/json writes a version as a JSON
// string. It returns no error.
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText sets v to the version that text states, as [Parse] reads it, and leaves v unchanged
// when Parse returns an error. It returns the error of Parse.
func (v *Version) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// IsZero reports whether v is 0.0.0 without a pre-release and without build metadata: the version
// of a package that has never been released.
func (v Version) IsZero() bool {
	return v == Version{}
}

// Compare returns -1 when v precedes w, 1 when w precedes v, and 0 when they have the same
// precedence, by the rules of Semantic Versioning 2.0.0: the numeric components in order, then a
// version with a pre-release before the same version without one, then the pre-release
// identifiers in order, a numeric one before an alphanumeric one. Build metadata plays no part.
func (v Version) Compare(w Version) int {
	if c := cmp.Compare(v.Major, w.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(v.Minor, w.Minor); c != 0 {
		return c
	}
	if c := cmp.Compare(v.Patch, w.Patch); c != 0 {
		return c
	}
	switch {
	case v.Pre == w.Pre:
		return 0
	case v.Pre == "":
		return 1
	case w.Pre == "":
		return -1
	}
	a, b := strings.Split(v.Pre, "."), strings.Split(w.Pre, ".")
	for i := range min(len(a), len(b)) {
		if c := comparePre(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

// Bump returns v released at level b, by the rules of the inc function of node-semver, and drops
// the build metadata:
//
//   - [BumpNone] returns v.
//   - [BumpPatch] increments the patch, or releases a pre-release of the same three components.
//   - [BumpMinor] increments the minor and resets the patch, or releases a pre-release of a version
//     whose patch is 0.
//   - [BumpMajor] increments the major and resets the minor and the patch, or releases a
//     pre-release of a version whose minor and patch are 0.
//
// It returns an error that wraps [ErrInvalid] for a b that is not valid, and one that wraps
// [ErrOverflow] for a component that would pass [MaxComponent].
func (v Version) Bump(b Bump) (Version, error) {
	if !b.Valid() {
		return Version{}, fmt.Errorf("%w: level %q, which is none of none, patch, minor and major", ErrInvalid, b)
	}
	if b == BumpNone {
		return v, nil
	}
	// A pre-release of a version releases as that version when the bump would reach it anyway,
	// so 2.0.0-rc.1 bumped at major is 2.0.0.
	pre := v.Pre != ""
	next := Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch}
	var incremented *uint64
	switch {
	case b == BumpMajor && (!pre || v.Minor != 0 || v.Patch != 0):
		next.Minor, next.Patch, incremented = 0, 0, &next.Major
	case b == BumpMinor && (!pre || v.Patch != 0):
		next.Patch, incremented = 0, &next.Minor
	case b == BumpPatch && !pre:
		incremented = &next.Patch
	}
	if incremented != nil {
		if *incremented >= MaxComponent {
			return Version{}, fmt.Errorf("%w: %s at level %s", ErrOverflow, v, b)
		}
		*incremented++
	}
	return next, nil
}

// component returns the numeric component s. It returns an error for an s that is empty, has a
// character other than a digit, has a leading zero, or is above [MaxComponent].
func component(s string) (uint64, error) {
	if !numeric(s) {
		return 0, fmt.Errorf("the component %q is not a number", s)
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, fmt.Errorf("the component %q has a leading zero", s)
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n > MaxComponent {
		return 0, fmt.Errorf("the component %s is above %d", s, MaxComponent)
	}
	return n, nil
}

// identifiers returns an error for a dot-separated list s with an identifier that is empty or has
// a character other than an ASCII letter, a digit and a hyphen. With pre set, it also returns an
// error for a numeric identifier with a leading zero, which the build metadata allows.
func identifiers(s string, pre bool) error {
	for id := range strings.SplitSeq(s, ".") {
		if id == "" {
			return fmt.Errorf("%q has an empty identifier", s)
		}
		for i := range len(id) {
			c := id[i]
			allowed := '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '-'
			if !allowed {
				return fmt.Errorf("%q has the character %q", s, c)
			}
		}
		if pre && numeric(id) && len(id) > 1 && id[0] == '0' {
			return fmt.Errorf("%q has the numeric identifier %s with a leading zero", s, id)
		}
	}
	return nil
}

// numeric reports whether s is a non-empty string of ASCII digits.
func numeric(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// comparePre compares two pre-release identifiers: numeric ones by their value, a numeric one
// before an alphanumeric one, and alphanumeric ones in the order of their ASCII bytes.
func comparePre(a, b string) int {
	na, nb := numeric(a), numeric(b)
	switch {
	case na && nb:
		return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
	case na:
		return -1
	case nb:
		return 1
	}
	return strings.Compare(a, b)
}
