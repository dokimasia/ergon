// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"fmt"
	"regexp"
	"strings"
)

// The characters of a requirement of PEP 508 that end or delimit its parts.
const (
	markerMark   = ";"
	urlMark      = "@"
	extrasOpen   = "["
	parensOpen   = "("
	parensClose  = ")"
	blanks       = " \t"
	nameSeparate = "-"
)

// head matches the name and the extras at the start of a requirement of PEP 508, with the name in
// its first group.
var head = regexp.MustCompile(`^\s*([A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?)(?:\s*\[[^\]]*\])?`)

// runs matches a run of the separators of a name, which the normal form of PEP 503 writes as one -.
var runs = regexp.MustCompile(`[-_.]+`)

// Requirement is a requirement of PEP 508, such as probe-a[cli]>=1.2; python_version >= "3.12", in
// the parts that a release reads and rewrites.
type Requirement struct {
	// Name is the name of the distribution as the requirement writes it.
	Name string

	// Specifier is the version specifier as the requirement writes it, without parentheses and the
	// spaces around it, such as >=1.2, and empty for a requirement without one.
	Specifier string

	// Start and End are the offsets of the specifier in the text of the requirement. A requirement
	// without a specifier has both after its name and its extras, where a specifier goes.
	Start, End int

	// URL reports a requirement of a URL, such as probe-a @ https://example.com/a.whl, which has no
	// specifier.
	URL bool
}

// ParseRequirement returns the requirement of PEP 508 that text states: its name, then its extras,
// then its specifier or its URL, and then its marker after a semicolon. It reads the specifier as
// text and does not parse it. It returns an error that wraps [ErrRequirement] for a text without a
// name, with extras that do not close, and with parentheses around its specifier that do not close.
func ParseRequirement(text string) (Requirement, error) {
	loc := head.FindStringSubmatchIndex(text)
	if loc == nil {
		return Requirement{}, fmt.Errorf("%w: %q, which does not start with a name", ErrRequirement, text)
	}
	r := Requirement{Name: text[loc[2]:loc[3]], Start: loc[1], End: loc[1]}
	spec, _, _ := strings.Cut(text[loc[1]:], markerMark)
	trimmed := strings.TrimSpace(spec)
	switch {
	case strings.HasPrefix(trimmed, extrasOpen):
		return Requirement{}, fmt.Errorf("%w: %q, whose extras do not close", ErrRequirement, text)
	case strings.HasPrefix(trimmed, urlMark):
		r.URL = true
		return r, nil
	case trimmed == "":
		return r, nil
	}
	start := loc[1] + len(spec) - len(strings.TrimLeft(spec, blanks))
	if strings.HasPrefix(trimmed, parensOpen) {
		if !strings.HasSuffix(trimmed, parensClose) {
			return Requirement{}, fmt.Errorf("%w: %q, whose parentheses do not close", ErrRequirement, text)
		}
		inner := trimmed[len(parensOpen) : len(trimmed)-len(parensClose)]
		start += len(parensOpen) + len(inner) - len(strings.TrimLeft(inner, blanks))
		trimmed = strings.TrimSpace(inner)
	}
	r.Specifier, r.Start, r.End = trimmed, start, start+len(trimmed)
	return r, nil
}

// Normalize returns name in the normal form of PEP 503, which a registry compares names in: in
// lowercase, with each run of -, _ and . as one -.
func Normalize(name string) string {
	return strings.ToLower(runs.ReplaceAllString(name, nameSeparate))
}
