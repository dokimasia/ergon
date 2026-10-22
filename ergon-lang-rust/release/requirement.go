// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/version"
)

// The operators of a comparator of a version requirement of Cargo, as the requirement writes them.
// A comparator without an operator is a caret comparator, and an exact one with a wildcard.
const (
	opExact        = "="
	opGreater      = ">"
	opGreaterEqual = ">="
	opLess         = "<"
	opLessEqual    = "<="
	opTilde        = "~"
	opCaret        = "^"
)

// The characters that separate the parts of a version requirement.
const (
	comparatorSeparator = ','
	numberSeparator     = '.'
	preSeparator        = '-'
	buildSeparator      = '+'
)

// space is the one character that a version requirement allows around its parts.
const space = ' '

// wildcards are the characters that stand for a number of a comparator, and alone for every
// version.
const wildcards = "*xX"

// digits are the characters of a number.
const digits = "0123456789"

// maxComparators is the most comparators of a version requirement, as the semver crate allows.
const maxComparators = 32

// operators are the operators of a comparator, each before the operators that it starts with.
var operators = []string{opGreaterEqual, opLessEqual, opExact, opGreater, opLess, opTilde, opCaret}

// comparator is a comparator of a version requirement of Cargo, such as ^1.2 or >=1.2.3-rc.1.
type comparator struct {
	// op is the operator as the requirement writes it, and empty for none.
	op string

	// pre is the pre-release of the version, and empty for none.
	pre string

	// major, minor and patch are the numbers of the version. minor and patch are 0 unless numbers
	// counts them.
	major, minor, patch uint64

	// numbers is the count of the numbers of the version before a wildcard: 1, 2 or 3.
	numbers int

	// start and end are the offsets of the version in the requirement: its numbers, its wildcards,
	// its pre-release and its build metadata. digits is the offset after its last number.
	start, digits, end int

	// wildcard reports a wildcard in place of the minor or the patch.
	wildcard bool
}

// requirement is a version requirement of Cargo, as the semver crate 1.0.28, the parser and the
// matcher of Cargo, reads it: a list of comparators that a version satisfies when it satisfies each
// of them.
type requirement struct {
	// comparators are the comparators in the order of the requirement. The wildcard of every
	// version has none.
	comparators []comparator

	// any reports the requirement of a dependency without a version, which every version satisfies.
	any bool
}

// parseRequirement returns the version requirement text, and the requirement of a dependency
// without a version for an empty text. It returns an error that wraps [ErrRequirement] for a text
// that the semver crate does not parse, such as a text with one of these:
//
//   - a comparator without a version
//   - a number with a leading zero or above the bound of a uint64
//   - a number after a wildcard
//   - an empty identifier of a pre-release or of build metadata
//   - a number of a pre-release with a leading zero
//   - two comparators without a comma between them
//   - a wildcard of every version beside other comparators
//   - more than 32 comparators
func parseRequirement(text string) (requirement, error) {
	if text == "" {
		return requirement{any: true}, nil
	}
	i := skip(text, 0)
	if i < len(text) && strings.IndexByte(wildcards, text[i]) >= 0 {
		if skip(text, i+1) < len(text) {
			return requirement{}, fmt.Errorf("%w: %q, whose wildcard is not the only comparator", ErrRequirement, text)
		}
		return requirement{}, nil
	}
	var r requirement
	for {
		c, next, err := parseComparator(text, i)
		if err != nil {
			return requirement{}, fmt.Errorf("%w: %q: %w", ErrRequirement, text, err)
		}
		r.comparators = append(r.comparators, c)
		switch {
		case next == len(text):
			return r, nil
		case text[next] != comparatorSeparator:
			return requirement{}, fmt.Errorf(
				"%w: %q, which has %q in place of a comma",
				ErrRequirement,
				text,
				text[next],
			)
		case len(r.comparators) == maxComparators:
			return requirement{}, fmt.Errorf("%w: %q, which has more than %d comparators", ErrRequirement, text,
				maxComparators)
		}
		i = skip(text, next+1)
	}
}

// parseComparator returns the comparator at the offset i of text, and the offset after it and the
// spaces after it. It returns an error for a comparator that the semver crate does not parse.
func parseComparator(text string, i int) (comparator, int, error) {
	var c comparator
	for _, op := range operators {
		if strings.HasPrefix(text[i:], op) {
			c.op, i = op, i+len(op)
			break
		}
	}
	i = skip(text, i)
	c.start = i
	var err error
	if c.major, i, err = number(text, i); err != nil {
		return comparator{}, 0, err
	}
	c.numbers, c.digits = 1, i
	for _, numbers := range []int{2, 3} {
		if i >= len(text) || text[i] != numberSeparator {
			break
		}
		i++
		switch {
		case i < len(text) && strings.IndexByte(wildcards, text[i]) >= 0:
			c.wildcard, i = true, i+1
			continue
		case c.wildcard:
			return comparator{}, 0, fmt.Errorf("the number %q after a wildcard", text[i:])
		}
		var n uint64
		if n, i, err = number(text, i); err != nil {
			return comparator{}, 0, err
		}
		if numbers == 2 {
			c.minor = n
		} else {
			c.patch = n
		}
		c.numbers, c.digits = numbers, i
	}
	if c.numbers == 3 && i < len(text) && text[i] == preSeparator {
		if c.pre, i, err = identifiers(text, i+1, true); err != nil {
			return comparator{}, 0, err
		}
	}
	if c.numbers == 3 && i < len(text) && text[i] == buildSeparator {
		if _, i, err = identifiers(text, i+1, false); err != nil {
			return comparator{}, 0, err
		}
	}
	c.end = i
	return c, skip(text, i), nil
}

// number returns the number at the offset i of text, and the offset after it. It returns an error
// for a text without a digit at i, a number with a leading zero and a number above the bound of a
// uint64.
func number(text string, i int) (uint64, int, error) {
	start := i
	var n uint64
	for ; i < len(text) && '0' <= text[i] && text[i] <= '9'; i++ {
		if i > start && n == 0 {
			return 0, 0, fmt.Errorf("the number %q, which has a leading zero", text[start:i+1])
		}
		d := uint64(text[i] - '0')
		if n > (math.MaxUint64-d)/10 {
			return 0, 0, fmt.Errorf("the number %q, which is above %d", text[start:i+1], uint64(math.MaxUint64))
		}
		n = n*10 + d
	}
	if i == start {
		return 0, 0, fmt.Errorf("the text %q in place of a number", text[start:])
	}
	return n, i, nil
}

// identifiers returns the dot-separated identifiers of letters, digits and hyphens at the offset i
// of text, and the offset after them. It returns an error for an empty identifier, and with pre for
// a number with a leading zero.
func identifiers(text string, i int, pre bool) (string, int, error) {
	start, segment := i, i
	for {
		for i < len(text) && (text[i] == '-' || '0' <= text[i] && text[i] <= '9' ||
			'a' <= text[i] && text[i] <= 'z' || 'A' <= text[i] && text[i] <= 'Z') {
			i++
		}
		id := text[segment:i]
		if id == "" {
			return "", 0, fmt.Errorf("an empty identifier in %q", text[start:])
		}
		if pre && len(id) > 1 && id[0] == '0' && strings.Trim(id, digits) == "" {
			return "", 0, fmt.Errorf("the identifier %q, which has a leading zero", id)
		}
		if i >= len(text) || text[i] != numberSeparator {
			return text[start:i], i, nil
		}
		i++
		segment = i
	}
}

// skip returns the offset of the first character of text from i that is not a space.
func skip(text string, i int) int {
	for i < len(text) && text[i] == space {
		i++
	}
	return i
}

// matches reports whether v satisfies r, by the rules of the semver crate: v satisfies each
// comparator of r, and a pre-release v also needs a comparator of r with a pre-release of the same
// numbers. Build metadata plays no part.
func (r *requirement) matches(v version.Version) bool {
	if r.any {
		return true
	}
	for i := range r.comparators {
		if !r.comparators[i].holds(v) {
			return false
		}
	}
	if v.Pre == "" {
		return true
	}
	for i := range r.comparators {
		c := &r.comparators[i]
		if c.numbers == 3 && c.pre != "" && c.major == v.Major && c.minor == v.Minor && c.patch == v.Patch {
			return true
		}
	}
	return false
}

// holds reports whether v satisfies c alone, by the rules of the operator of c in the semver crate.
func (c *comparator) holds(v version.Version) bool {
	switch {
	case c.op == opExact || c.op == "" && c.wildcard:
		return c.exact(v)
	case c.op == opGreater:
		return c.greater(v)
	case c.op == opGreaterEqual:
		return c.exact(v) || c.greater(v)
	case c.op == opLess:
		return c.less(v)
	case c.op == opLessEqual:
		return c.exact(v) || c.less(v)
	case c.op == opTilde:
		return c.tilde(v)
	}
	return c.caret(v)
}

// exact reports whether v has the numbers that c has and the pre-release of c.
func (c *comparator) exact(v version.Version) bool {
	return v.Major == c.major && (c.numbers < 2 || v.Minor == c.minor) && (c.numbers < 3 || v.Patch == c.patch) &&
		v.Pre == c.pre
}

// greater reports whether v follows the version of c in the numbers that c has, and in the
// pre-release when c has three numbers.
func (c *comparator) greater(v version.Version) bool {
	switch {
	case v.Major != c.major:
		return v.Major > c.major
	case c.numbers < 2:
		return false
	case v.Minor != c.minor:
		return v.Minor > c.minor
	case c.numbers < 3:
		return false
	case v.Patch != c.patch:
		return v.Patch > c.patch
	}
	return version.Version{Pre: v.Pre}.Compare(version.Version{Pre: c.pre}) > 0
}

// less reports whether v precedes the version of c in the numbers that c has, and in the
// pre-release when c has three numbers.
func (c *comparator) less(v version.Version) bool {
	switch {
	case v.Major != c.major:
		return v.Major < c.major
	case c.numbers < 2:
		return false
	case v.Minor != c.minor:
		return v.Minor < c.minor
	case c.numbers < 3:
		return false
	case v.Patch != c.patch:
		return v.Patch < c.patch
	}
	return version.Version{Pre: v.Pre}.Compare(version.Version{Pre: c.pre}) < 0
}

// tilde reports whether v satisfies the tilde comparator c: the major and the minor of c, and at
// least its patch and pre-release.
func (c *comparator) tilde(v version.Version) bool {
	switch {
	case v.Major != c.major:
		return false
	case c.numbers >= 2 && v.Minor != c.minor:
		return false
	case c.numbers >= 3 && v.Patch != c.patch:
		return v.Patch > c.patch
	}
	return version.Version{Pre: v.Pre}.Compare(version.Version{Pre: c.pre}) >= 0
}

// caret reports whether v satisfies the caret comparator c: the first number of c that is not 0,
// or its last number, and at least the version of c.
func (c *comparator) caret(v version.Version) bool {
	switch {
	case v.Major != c.major:
		return false
	case c.numbers < 2:
		return true
	case c.numbers < 3 && c.major > 0:
		return v.Minor >= c.minor
	case c.numbers < 3:
		return v.Minor == c.minor
	case c.major > 0 && v.Minor != c.minor:
		return v.Minor > c.minor
	case c.major > 0 && v.Patch != c.patch:
		return v.Patch > c.patch
	case c.major == 0 && c.minor > 0 && v.Minor != c.minor:
		return false
	case c.major == 0 && c.minor > 0 && v.Patch != c.patch:
		return v.Patch > c.patch
	case c.major == 0 && c.minor == 0 && (v.Minor != c.minor || v.Patch != c.patch):
		return false
	}
	return version.Version{Pre: v.Pre}.Compare(version.Version{Pre: c.pre}) >= 0
}

// moved returns the version of c, a comparator of req, moved to admit v with the operator of c, as
// [Versioner.Rewrite] states it. It returns an error that wraps [ErrRequirement] for a comparator
// of > that excludes v.
func (c *comparator) moved(req string, v version.Version) (string, error) {
	suffix := ""
	if c.wildcard {
		suffix = req[c.digits:c.end]
	}
	numbers := []uint64{v.Major, v.Minor, v.Patch}[:c.numbers]
	switch {
	case (c.op == opGreater || c.op == opLess || c.op == opLessEqual) && c.holds(v):
		return req[c.start:c.end], nil
	case c.op == opGreater:
		return "", fmt.Errorf("%w: %q, whose comparator %s excludes %s", ErrRequirement, req, req[c.start:c.end], v)
	case c.op == opLess:
		numbers[len(numbers)-1]++
	case v.Pre != "" && !c.wildcard:
		return version.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch, Pre: v.Pre}.String(), nil
	}
	parts := make([]string, len(numbers))
	for i, n := range numbers {
		parts[i] = strconv.FormatUint(n, 10)
	}
	return strings.Join(parts, string(numberSeparator)) + suffix, nil
}
