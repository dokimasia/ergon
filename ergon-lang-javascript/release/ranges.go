// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/version"
)

// The sources of the regular expressions of the ranges of node-semver 7.8.5, as its strict mode
// reads them.
const (
	srcNumeric      = `0|[1-9]\d*`
	srcNumericLoose = `\d+`
	srcNonNumeric   = `\d*[a-zA-Z-][a-zA-Z0-9-]*`
	srcMain         = `(` + srcNumeric + `)\.(` + srcNumeric + `)\.(` + srcNumeric + `)`
	srcMainLoose    = `(` + srcNumericLoose + `)\.(` + srcNumericLoose + `)\.(` + srcNumericLoose + `)`
	srcPreID        = `(?:` + srcNonNumeric + `|` + srcNumeric + `)`
	srcPreIDLoose   = `(?:` + srcNonNumeric + `|` + srcNumericLoose + `)`
	srcPre          = `(?:-(` + srcPreID + `(?:\.` + srcPreID + `)*))`
	srcPreLoose     = `(?:-?(` + srcPreIDLoose + `(?:\.` + srcPreIDLoose + `)*))`
	srcBuildID      = `[a-zA-Z0-9-]+`
	srcBuild        = `(?:\+(` + srcBuildID + `(?:\.` + srcBuildID + `)*))`
	srcFullPlain    = `v?` + srcMain + srcPre + `?` + srcBuild + `?`
	srcLoosePlain   = `[v=\s]*` + srcMainLoose + srcPreLoose + `?` + srcBuild + `?`
	srcOperator     = `((?:<|>)?=?)`
	srcXID          = srcNumeric + `|x|X|\*`
	srcXRangePlain  = `[v=\s]*(` + srcXID + `)(?:\.(` + srcXID + `)(?:\.(` + srcXID + `)(?:` + srcPre + `)?` +
		srcBuild + `?)?)?`
)

// The operators of a comparator of node-semver.
const (
	opEqual        = "="
	opLess         = "<"
	opLessEqual    = "<="
	opGreater      = ">"
	opGreaterEqual = ">="
)

// The parts of a range of node-semver that the desugaring writes.
const (
	// nullSet is the comparator that every version fails.
	nullSet = "<0.0.0-0"

	// anyVersion is the x-range that every version passes, which the desugaring then drops.
	anyVersion = "*"

	// zeroComponent is a component of 0.
	zeroComponent = "0"

	// lowestPre is the pre-release that precedes every other pre-release of a version.
	lowestPre = "-0"

	// alternatives separates the sets of a range.
	alternatives = "||"
)

// xComponents are the spellings of a component of an x-range that admits every value: an absent
// component, x, X and *.
var xComponents = []string{"", "x", "X", anyVersion}

// The regular expressions of the ranges of node-semver 7.8.5 in its strict mode, from the sources
// of internal/re.js.
var (
	reBuild        = regexp.MustCompile(srcBuild)
	reHyphen       = regexp.MustCompile(`^\s*(` + srcXRangePlain + `)\s+-\s+(` + srcXRangePlain + `)\s*$`)
	reOperatorTrim = regexp.MustCompile(`(\s*)` + srcOperator + `\s*(` + srcLoosePlain + `|` + srcXRangePlain + `)`)
	reTildeTrim    = regexp.MustCompile(`(\s*)(?:~>?)\s+`)
	reCaretTrim    = regexp.MustCompile(`(\s*)\^\s+`)
	reTilde        = regexp.MustCompile(`^(?:~>?)` + srcXRangePlain + `$`)
	reCaret        = regexp.MustCompile(`^\^` + srcXRangePlain + `$`)
	reXRange       = regexp.MustCompile(`^` + srcOperator + `\s*` + srcXRangePlain + `$`)
	reStar         = regexp.MustCompile(`(<|>)?=?\s*\*`)
	reLowestBound  = regexp.MustCompile(`^\s*>=\s*0\.0\.0\s*$`)
	reComparator   = regexp.MustCompile(`^` + srcOperator + `\s*(` + srcFullPlain + `)$|^$`)
	reSpaces       = regexp.MustCompile(`\s+`)
)

// comparator is a comparison of a version with a version of a range of node-semver, or any version.
type comparator struct {
	// op is the operator: "", which is equality, <, <=, > or >=.
	op string

	// v is the version of the comparison, without build metadata.
	v version.Version

	// any reports the comparator that every version passes, which node-semver writes as "".
	any bool
}

// value returns c as node-semver writes it: its operator and its version, and "" for any version.
func (c *comparator) value() string {
	if c.any {
		return ""
	}
	return c.op + c.v.String()
}

// test reports whether v passes c. Build metadata plays no part.
func (c *comparator) test(v version.Version) bool {
	if c.any {
		return true
	}
	n := v.Compare(c.v)
	switch c.op {
	case opLess:
		return n < 0
	case opLessEqual:
		return n <= 0
	case opGreater:
		return n > 0
	case opGreaterEqual:
		return n >= 0
	}
	return n == 0
}

// semverRange is a range of node-semver: a version satisfies it when the version satisfies one of
// its sets of comparators, as [testSet] states.
type semverRange [][]comparator

// parseRange returns the range that s states, as the Range of node-semver 7.8.5 reads it in its
// strict mode, and reports whether s states one. A range of two or more sets drops each set that no
// version satisfies, and is the set of any version alone when one of its sets is.
func parseRange(s string) (semverRange, bool) {
	raw := strings.Join(strings.Fields(s), " ")
	var r semverRange
	for part := range strings.SplitSeq(raw, alternatives) {
		set, ok := parseSet(strings.TrimSpace(part))
		if !ok {
			return nil, false
		}
		r = append(r, set)
	}
	if len(r) == 1 {
		return r, true
	}
	first := r[0]
	r = slices.DeleteFunc(r, func(set []comparator) bool { return set[0].value() == nullSet })
	if len(r) == 0 {
		return semverRange{first}, true
	}
	if i := slices.IndexFunc(r, func(set []comparator) bool { return len(set) == 1 && set[0].any }); i >= 0 {
		return semverRange{r[i]}, true
	}
	return r, true
}

// testSet reports whether v passes every comparator of set, as the testSet of node-semver: a
// pre-release of v also needs a comparator of set whose version is a pre-release of the major, the
// minor and the patch of v.
func testSet(set []comparator, v version.Version) bool {
	for i := range set {
		if !set[i].test(v) {
			return false
		}
	}
	if v.Pre == "" {
		return true
	}
	return slices.ContainsFunc(set, func(c comparator) bool {
		return !c.any && c.v.Pre != "" && c.v.Major == v.Major && c.v.Minor == v.Minor && c.v.Patch == v.Patch
	})
}

// parseSet returns the comparators of the set s of a range, as the parseRange of node-semver reads
// a set, and reports whether s states one. It drops the build metadata, and desugars a hyphen range,
// carets, tildes, x-ranges and stars into comparators. It keeps each comparator once. A set with
// the comparator that every version fails is that comparator alone. A set of two or more
// comparators drops the comparator of any version.
func parseSet(s string) ([]comparator, bool) {
	s = reBuild.ReplaceAllString(s, "")
	if m := reHyphen.FindStringSubmatch(s); m != nil {
		s = hyphen(m)
	}
	s = reOperatorTrim.ReplaceAllString(s, "${1}${2}${3}")
	s = reTildeTrim.ReplaceAllString(s, "${1}~")
	s = reCaretTrim.ReplaceAllString(s, "${1}^")
	parts := strings.Split(s, " ")
	for i, part := range parts {
		parts[i] = stars(each(each(each(part, caret), tilde), xRange))
	}
	var set []comparator
	for _, token := range reSpaces.Split(strings.Join(parts, " "), -1) {
		c, ok := newComparator(reLowestBound.ReplaceAllString(strings.TrimSpace(token), ""))
		if !ok {
			return nil, false
		}
		set = append(set, c)
	}
	if i := slices.IndexFunc(set, func(c comparator) bool { return c.value() == nullSet }); i >= 0 {
		return set[i : i+1], true
	}
	var out []comparator
	for i := range set {
		value := set[i].value()
		if !slices.ContainsFunc(out, func(o comparator) bool { return o.value() == value }) {
			out = append(out, set[i])
		}
	}
	if len(out) > 1 {
		out = slices.DeleteFunc(out, func(c comparator) bool { return c.any })
	}
	return out, true
}

// newComparator returns the comparator that s states, as the Comparator of node-semver reads it,
// and reports whether s states one: an operator and a version with an optional v, or nothing for any
// version.
func newComparator(s string) (comparator, bool) {
	m := reComparator.FindStringSubmatch(strings.Join(strings.Fields(s), " "))
	if m == nil {
		return comparator{}, false
	}
	if m[2] == "" {
		return comparator{any: true}, true
	}
	v, err := version.Parse(strings.TrimPrefix(m[2], "v"))
	if err != nil {
		return comparator{}, false
	}
	op := m[1]
	if op == opEqual {
		op = ""
	}
	return comparator{op: op, v: version.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch, Pre: v.Pre}}, true
}

// each returns the comparators of s, separated by spaces, each replaced by f, as node-semver
// desugars each comparator of a set.
func each(s string, f func(string) string) string {
	parts := reSpaces.Split(strings.TrimSpace(s), -1)
	for i, part := range parts {
		parts[i] = f(part)
	}
	return strings.Join(parts, " ")
}

// hyphen returns the comparators of a hyphen range, such as >=1.2.3 <=2.3.4 for 1.2.3 - 2.3.4, from
// the submatches m of the range: the lower version in m[1] with its components in m[2] to m[4], and
// the upper version in m[7] with its components and pre-release in m[8] to m[11].
func hyphen(m []string) string {
	lower, lowerMajor, lowerMinor, lowerPatch := m[1], m[2], m[3], m[4]
	upper, upperMajor, upperMinor, upperPatch, upperPre := m[7], m[8], m[9], m[10], m[11]
	switch {
	case slices.Contains(xComponents, lowerMajor):
		lower = ""
	case slices.Contains(xComponents, lowerMinor):
		lower = opGreaterEqual + lowerMajor + ".0.0"
	case slices.Contains(xComponents, lowerPatch):
		lower = opGreaterEqual + lowerMajor + "." + lowerMinor + ".0"
	default:
		lower = opGreaterEqual + lower
	}
	switch {
	case slices.Contains(xComponents, upperMajor):
		upper = ""
	case slices.Contains(xComponents, upperMinor):
		upper = opLess + next(upperMajor) + ".0.0" + lowestPre
	case slices.Contains(xComponents, upperPatch):
		upper = opLess + upperMajor + "." + next(upperMinor) + ".0" + lowestPre
	case upperPre != "":
		upper = opLessEqual + upperMajor + "." + upperMinor + "." + upperPatch + "-" + upperPre
	default:
		upper = opLessEqual + upper
	}
	return strings.TrimSpace(lower + " " + upper)
}

// caret returns the comparators of a caret range, such as >=1.2.3 <2.0.0-0 for ^1.2.3, and c for any
// other comparator. A caret admits the versions up to the next change of its first component that
// is not 0.
func caret(c string) string {
	m := reCaret.FindStringSubmatch(c)
	if m == nil {
		return c
	}
	major, minor, patch, pre := m[1], m[2], m[3], m[4]
	switch {
	case slices.Contains(xComponents, major):
		return ""
	case slices.Contains(xComponents, minor):
		return opGreaterEqual + major + ".0.0 " + opLess + next(major) + ".0.0" + lowestPre
	case slices.Contains(xComponents, patch) && major == zeroComponent:
		return opGreaterEqual + major + "." + minor + ".0 " + opLess + major + "." + next(minor) + ".0" + lowestPre
	case slices.Contains(xComponents, patch):
		return opGreaterEqual + major + "." + minor + ".0 " + opLess + next(major) + ".0.0" + lowestPre
	}
	lower := opGreaterEqual + major + "." + minor + "." + patch
	if pre != "" {
		lower += "-" + pre
	}
	switch {
	case major != zeroComponent:
		return lower + " " + opLess + next(major) + ".0.0" + lowestPre
	case minor != zeroComponent:
		return lower + " " + opLess + major + "." + next(minor) + ".0" + lowestPre
	}
	return lower + " " + opLess + major + "." + minor + "." + next(patch) + lowestPre
}

// tilde returns the comparators of a tilde range, such as >=1.2.3 <1.3.0-0 for ~1.2.3, and c for any
// other comparator. A tilde admits the patches of its minor, or the minors of its major when it
// states no minor.
func tilde(c string) string {
	m := reTilde.FindStringSubmatch(c)
	if m == nil {
		return c
	}
	major, minor, patch, pre := m[1], m[2], m[3], m[4]
	switch {
	case slices.Contains(xComponents, major):
		return ""
	case slices.Contains(xComponents, minor):
		return opGreaterEqual + major + ".0.0 " + opLess + next(major) + ".0.0" + lowestPre
	case slices.Contains(xComponents, patch):
		return opGreaterEqual + major + "." + minor + ".0 " + opLess + major + "." + next(minor) + ".0" + lowestPre
	}
	lower := opGreaterEqual + major + "." + minor + "." + patch
	if pre != "" {
		lower += "-" + pre
	}
	return lower + " " + opLess + major + "." + next(minor) + ".0" + lowestPre
}

// xRange returns the comparators of an x-range, such as >=1.2.0 <1.3.0-0 for 1.2.x and <2.0.0-0 for
// <=1, and c for any other comparator. A comparator whose x precedes a number, such as x.1, stays as
// it is, and is not valid.
func xRange(c string) string {
	c = strings.TrimSpace(c)
	m := reXRange.FindStringSubmatch(c)
	if m == nil {
		return c
	}
	op, major, minor, patch := m[1], m[2], m[3], m[4]
	anyMinor, anyPatch := slices.Contains(xComponents, minor), slices.Contains(xComponents, patch)
	xMajor := slices.Contains(xComponents, major)
	if (xMajor && !anyMinor) || (anyMinor && patch != "" && !anyPatch) {
		return c
	}
	xMinor := xMajor || anyMinor
	xPatch := xMinor || anyPatch
	if op == opEqual && xPatch {
		op = ""
	}
	switch {
	case xMajor && (op == opGreater || op == opLess):
		return nullSet
	case xMajor:
		return anyVersion
	case op != "" && xPatch:
		return bound(op, major, minor, xMinor)
	case xMinor:
		return opGreaterEqual + major + ".0.0 " + opLess + next(major) + ".0.0" + lowestPre
	case xPatch:
		return opGreaterEqual + major + "." + minor + ".0 " + opLess + major + "." + next(minor) + ".0" + lowestPre
	}
	return c
}

// bound returns the comparator of an x-range with the operator op and an x after major, or after
// minor when xMinor is false: >1.x is >=2.0.0, >1.2.x is >=1.3.0, <=1.x is <2.0.0-0, <1.x is
// <1.0.0-0, and >=1.x is >=1.0.0.
func bound(op, major, minor string, xMinor bool) string {
	if xMinor {
		minor = zeroComponent
	}
	switch op {
	case opGreater:
		op = opGreaterEqual
		if xMinor {
			major = next(major)
		} else {
			minor = next(minor)
		}
	case opLessEqual:
		op = opLess
		if xMinor {
			major = next(major)
		} else {
			minor = next(minor)
		}
	}
	out := op + major + "." + minor + ".0"
	if op == opLess {
		out += lowestPre
	}
	return out
}

// stars returns c without its first star and the operator before it, as node-semver drops the
// stars of a set.
func stars(c string) string {
	c = strings.TrimSpace(c)
	if loc := reStar.FindStringIndex(c); loc != nil {
		return c[:loc[0]] + c[loc[1]:]
	}
	return c
}

// next returns the component n plus one, as node-semver adds one to a component. A component above
// [version.MaxComponent] stays as it is, since the comparator that it ends in is not valid either
// way.
func next(n string) string {
	v, err := strconv.ParseUint(n, 10, 64)
	if err != nil || v > version.MaxComponent {
		return n
	}
	return strconv.FormatUint(v+1, 10)
}
