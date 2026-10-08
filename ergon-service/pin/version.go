// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/module"
)

// The forms of the versions that a resolver orders. A number of at most 19 digits fits 64 bits.
var (
	// stableVersion matches a stable version: one to four numbers of at most 19 digits separated by
	// dots, with an optional leading v, such as 2.14.0 or v7.0.1.
	stableVersion = regexp.MustCompile(`^v?[0-9]{1,19}(\.[0-9]{1,19}){0,3}$`)

	// leadingNumbers matches the numbers at the start of a version, each of at most 19 digits, with
	// an optional leading v, such as v2.0.0 of v2.0.0-rc.1.
	leadingNumbers = regexp.MustCompile(`^v?[0-9]{1,19}(\.[0-9]{1,19})*`)
)

// rank is the position of a version among the versions of a project: by its numbers, then a release
// above a pre-release of the same numbers, then a pseudo-version of Go by the time of its commit. A
// pseudo-version is a pre-release of its numbers, so v1.2.4-0.20191109021931-daa7c04131f5 ranks
// below v1.2.4 and above v1.2.3.
type rank struct {
	// commit is the time of the commit of a pseudo-version of Go, and the zero time for any other
	// version.
	commit time.Time

	// numbers are the numbers at the start of the version, at least one, such as 2, 14 and 0 of
	// v2.14.0. The first is the major version.
	numbers []uint64

	// prerelease reports a version with text after its numbers, such as v2.0.0-rc.1 or a
	// pseudo-version.
	prerelease bool
}

// order returns the rank of the version v: the numbers at its start, which need not form a stable
// version, such as 2, 0 and 0 of v2.0.0-rc.1, whether text follows them, and the time of the commit
// of a pseudo-version of Go. A version that does not start with a number ranks as the version 0.
func order(v string) rank {
	prefix := leadingNumbers.FindString(v)
	var numbers []uint64
	for part := range strings.SplitSeq(strings.TrimPrefix(prefix, "v"), ".") {
		// A part of at most 19 digits parses, and the empty part of a version without numbers is 0.
		n, _ := strconv.ParseUint(part, 10, 64)
		numbers = append(numbers, n)
	}
	commit, _ := module.PseudoVersionTime(v)
	return rank{commit: commit, numbers: numbers, prerelease: prefix != v}
}

// above reports whether r ranks above s, as [rank] states. A missing number counts as 0, so v1.2
// ranks with v1.2.0.
func (r rank) above(s rank) bool {
	for i := range max(len(r.numbers), len(s.numbers)) {
		var a, b uint64
		if i < len(r.numbers) {
			a = r.numbers[i]
		}
		if i < len(s.numbers) {
			b = s.numbers[i]
		}
		if a > b {
			return true
		}
		if a < b {
			return false
		}
	}
	if r.prerelease != s.prerelease {
		return s.prerelease
	}
	return r.commit.After(s.commit)
}

// like returns version in the form of the version of a pin: with a leading v when pin has one, and
// without one when pin has none, as the tag v0.12.0 of commitlint is the version 0.12.0 of its pin.
func like(pin, version string) string {
	v := strings.TrimPrefix(version, "v")
	if strings.HasPrefix(pin, "v") {
		return "v" + v
	}
	return v
}
