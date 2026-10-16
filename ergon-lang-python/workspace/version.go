// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/version"
)

// The labels of a pre-release of PEP 440, in their order.
const (
	labelAlpha     = "a"
	labelBeta      = "b"
	labelCandidate = "rc"
)

// The parts of the normal form of a version of PEP 440 that are not numbers.
const (
	postPart  = ".post"
	devPart   = ".dev"
	epochMark = "!"
	localMark = "+"
)

// labels are the labels of a pre-release in their order.
var labels = []string{labelAlpha, labelBeta, labelCandidate}

// pattern matches a version of PEP 440, as the pattern of its appendix B, with surrounding spaces.
// Its groups are the epoch, the release, the label and the number of the pre-release, the number
// of a post-release of the form -N, the word and the number of a post-release of the other forms,
// the development release and its number, and the local label.
var pattern = regexp.MustCompile(`(?i)^\s*v?(?:([0-9]+)!)?([0-9]+(?:\.[0-9]+)*)` +
	`(?:[-_.]?(alpha|a|beta|b|preview|pre|c|rc)[-_.]?([0-9]+)?)?` +
	`(?:-([0-9]+)|[-_.]?(post|rev|r)[-_.]?([0-9]+)?)?` +
	`([-_.]?dev[-_.]?([0-9]+)?)?` +
	`(?:\+([a-z0-9]+(?:[-_.][a-z0-9]+)*))?\s*$`)

// separators matches the separators of a local label, which the normal form writes as dots.
var separators = regexp.MustCompile(`[-_]`)

// preLabels maps each spelling of a label of a pre-release to the label of the normal form.
var preLabels = map[string]string{
	"alpha": labelAlpha, labelAlpha: labelAlpha, "beta": labelBeta, labelBeta: labelBeta,
	"preview": labelCandidate, "pre": labelCandidate, "c": labelCandidate, labelCandidate: labelCandidate,
}

// semverPre matches the pre-release of a version of Semantic Versioning that PEP 440 has: a label
// and a number, such as rc.1.
var semverPre = regexp.MustCompile(`^(a|b|rc)\.(0|[1-9][0-9]*)$`)

// Version is a version of PEP 440, such as 1.2.0rc1, in the parts that its order compares. The
// zero value is version 0.
type Version struct {
	// Pre is the label of the pre-release, a, b or rc, and empty for a version without one.
	Pre string

	// Local is the local label in lowercase with dots between its parts, such as ubuntu.1, and
	// empty for a version without one.
	Local string

	// Release are the numbers of the release, such as 1, 2 and 0 of 1.2.0.
	Release []uint64

	// Epoch is the epoch, which a version states before an exclamation mark, and 0 for none.
	Epoch uint64

	// PreNumber is the number of the pre-release.
	PreNumber uint64

	// Post is the number of the post-release, and Dev the number of the development release. Each
	// counts only when its Has field is true.
	Post, Dev uint64

	// HasPost and HasDev report a post-release and a development release.
	HasPost, HasDev bool
}

// ParseVersion returns the version of PEP 440 that text states, with the spellings that PEP 440
// allows, such as 1.0-Alpha.1 for 1.0a1. A pre-release, a post-release or a development release
// without a number has the number 0. It returns an error that wraps [ErrVersion] for a text that is
// not a version, and for a number above the range of uint64.
func ParseVersion(text string) (Version, error) {
	m := pattern.FindStringSubmatch(text)
	if m == nil {
		return Version{}, fmt.Errorf("%w: %q, which is not a version of PEP 440", ErrVersion, text)
	}
	var v Version
	numbers := []struct {
		dst  *uint64
		text string
	}{
		{dst: &v.Epoch, text: m[1]},
		{dst: &v.PreNumber, text: m[4]},
		{dst: &v.Post, text: m[5] + m[7]},
		{dst: &v.Dev, text: m[9]},
	}
	for _, n := range numbers {
		if n.text == "" {
			continue
		}
		parsed, err := strconv.ParseUint(n.text, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %q: %w", ErrVersion, text, err)
		}
		*n.dst = parsed
	}
	for part := range strings.SplitSeq(m[2], ".") {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %q: %w", ErrVersion, text, err)
		}
		v.Release = append(v.Release, n)
	}
	v.Pre = preLabels[strings.ToLower(m[3])]
	v.HasPost = m[5] != "" || m[6] != ""
	v.HasDev = m[8] != ""
	v.Local = strings.ToLower(separators.ReplaceAllString(m[10], "."))
	return v, nil
}

// String returns v in the normal form of PEP 440, such as 1!1.2.0rc1.post2.dev3+local.1.
func (v Version) String() string {
	var b strings.Builder
	if v.Epoch != 0 {
		b.WriteString(strconv.FormatUint(v.Epoch, 10))
		b.WriteString(epochMark)
	}
	for i, n := range v.Release {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.FormatUint(n, 10))
	}
	if v.Pre != "" {
		b.WriteString(v.Pre)
		b.WriteString(strconv.FormatUint(v.PreNumber, 10))
	}
	if v.HasPost {
		b.WriteString(postPart)
		b.WriteString(strconv.FormatUint(v.Post, 10))
	}
	if v.HasDev {
		b.WriteString(devPart)
		b.WriteString(strconv.FormatUint(v.Dev, 10))
	}
	if v.Local != "" {
		b.WriteString(localMark)
		b.WriteString(v.Local)
	}
	return b.String()
}

// Compare returns -1 when v precedes w, 1 when w precedes v, and 0 when they are the same version,
// by the order of PEP 440: the epoch, the release without its trailing zeros, the pre-release, the
// post-release, the development release and the local label. A development release of a version
// without a pre-release or a post-release precedes its pre-releases.
func (v *Version) Compare(w *Version) int {
	if c := cmp.Compare(v.Epoch, w.Epoch); c != 0 {
		return c
	}
	if c := slices.Compare(trimZeros(v.Release), trimZeros(w.Release)); c != 0 {
		return c
	}
	if c := cmp.Compare(v.preRank(), w.preRank()); c != 0 {
		return c
	}
	if c := cmp.Compare(v.PreNumber, w.PreNumber); c != 0 && v.Pre != "" {
		return c
	}
	if c := compareOptional(v.HasPost, v.Post, w.HasPost, w.Post, -1); c != 0 {
		return c
	}
	if c := compareOptional(v.HasDev, v.Dev, w.HasDev, w.Dev, 1); c != 0 {
		return c
	}
	return compareLocal(v.Local, w.Local)
}

// IsPre reports whether v is a pre-release or a development release, which PEP 440 calls a
// pre-release.
func (v *Version) IsPre() bool {
	return v.Pre != "" || v.HasDev
}

// Base returns the epoch and the release of v, without its pre-release, its post-release, its
// development release and its local label.
func (v *Version) Base() Version {
	return Version{Epoch: v.Epoch, Release: v.Release}
}

// SemVer returns v as a version of Semantic Versioning 2.0.0: its release with a 0 for each
// missing number, such as 1.2.0 for 1.2, and its pre-release as the label and the number, such as
// rc.1 for rc1. It returns an error that wraps [ErrVersion] for a v that Semantic Versioning does
// not have: an epoch, a post-release, a development release, a local label, a release of more than
// three numbers, and a number above [version.MaxComponent].
func (v *Version) SemVer() (version.Version, error) {
	if v.Epoch != 0 || v.HasPost || v.HasDev || v.Local != "" || len(v.Release) > 3 {
		return version.Version{}, fmt.Errorf("%w: %s, which Semantic Versioning does not have", ErrVersion, v)
	}
	var numbers [3]uint64
	copy(numbers[:], v.Release)
	text := fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2])
	if v.Pre != "" {
		text += "-" + v.Pre + "." + strconv.FormatUint(v.PreNumber, 10)
	}
	s, err := version.Parse(text)
	if err != nil {
		return version.Version{}, fmt.Errorf("%w: %s: %w", ErrVersion, v, err)
	}
	return s, nil
}

// FromSemVer returns the version of PEP 440 of v, such as 1.2.0rc1 for 1.2.0-rc.1. It returns an
// error that wraps [ErrVersion] for build metadata, for a pre-release other than a label of PEP 440
// and a number, such as a.1, b.2 or rc.3, and for a number of a pre-release above the range of
// uint64.
func FromSemVer(v version.Version) (Version, error) {
	if v.Build != "" {
		return Version{}, fmt.Errorf("%w: %s, whose build metadata PEP 440 does not have", ErrVersion, v)
	}
	out := Version{Release: []uint64{v.Major, v.Minor, v.Patch}}
	if v.Pre == "" {
		return out, nil
	}
	m := semverPre.FindStringSubmatch(v.Pre)
	if m == nil {
		return Version{}, fmt.Errorf("%w: %s, whose pre-release is none of a.N, b.N and rc.N", ErrVersion, v)
	}
	n, err := strconv.ParseUint(m[2], 10, 64)
	if err != nil {
		return Version{}, fmt.Errorf("%w: %s: %w", ErrVersion, v, err)
	}
	out.Pre, out.PreNumber = m[1], n
	return out, nil
}

// preRank returns the place of the pre-release of v in the order of PEP 440: 0 for a development
// release of a version without a pre-release and a post-release, the place of its label from 1, and
// the place after every label for a version without a pre-release.
func (v *Version) preRank() int {
	if v.Pre == "" && !v.HasPost && v.HasDev {
		return 0
	}
	if v.Pre == "" {
		return len(labels) + 1
	}
	return slices.Index(labels, v.Pre) + 1
}

// trimZeros returns release without its trailing zeros, as PEP 440 compares a release.
func trimZeros(release []uint64) []uint64 {
	end := len(release)
	for end > 0 && release[end-1] == 0 {
		end--
	}
	return release[:end]
}

// compareOptional compares two numbers that each count only when its has is true. A missing number
// precedes every number when missing is -1, and follows every number when it is 1.
func compareOptional(hasA bool, a uint64, hasB bool, b uint64, missing int) int {
	switch {
	case hasA && hasB:
		return cmp.Compare(a, b)
	case hasA == hasB:
		return 0
	case hasA:
		return -missing
	}
	return missing
}

// compareLocal compares two local labels by the order of PEP 440: a version without a label first,
// then each part in order, a part of letters before a part of digits, and a label before a longer
// label that starts with it.
func compareLocal(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return -1
	case b == "":
		return 1
	}
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(x), len(y)) {
		if c := compareLocalPart(x[i], y[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(x), len(y))
}

// compareLocalPart compares two parts of a local label: two numbers by their value, two parts of
// letters by their bytes, and a part of letters before a number.
func compareLocalPart(a, b string) int {
	m, errA := strconv.ParseUint(a, 10, 64)
	n, errB := strconv.ParseUint(b, 10, 64)
	switch {
	case errA == nil && errB == nil:
		return cmp.Compare(m, n)
	case errA == nil:
		return 1
	case errB == nil:
		return -1
	}
	return strings.Compare(a, b)
}
