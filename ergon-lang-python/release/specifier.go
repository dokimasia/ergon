// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"fmt"
	"strconv"
	"strings"

	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// The operators of a version specifier of PEP 440.
const (
	opArbitrary    = "==="
	opCompatible   = "~="
	opEqual        = "=="
	opNotEqual     = "!="
	opLessEqual    = "<="
	opGreaterEqual = ">="
	opLess         = "<"
	opGreater      = ">"
)

// The separators of a version specifier: between its clauses, and between the numbers of a
// release.
const (
	clauseSeparator = ","
	numberSeparator = "."
)

// wildcard is the suffix of the version of a clause of == or != that matches each version with the
// numbers before it.
const wildcard = ".*"

// operators are the operators of a clause, each before the operators that it starts with.
var operators = []string{
	opArbitrary, opCompatible, opEqual, opNotEqual, opLessEqual, opGreaterEqual, opLess, opGreater,
}

// clause is one clause of a version specifier of PEP 440, such as >=1.2.0 or ==1.2.*.
type clause struct {
	// op is the operator of the clause.
	op string

	// raw is the clause as the specifier writes it, without the spaces around it.
	raw string

	// text is the version as the clause writes it, without a wildcard.
	text string

	// version is the version of the clause, without a wildcard. A clause of === has none.
	version pyworkspace.Version

	// start and end are the offsets of raw in the specifier.
	start, end int

	// prefix reports a clause of == or != with a wildcard.
	prefix bool
}

// parseSpecifier returns the clauses of the version specifier spec, in its order, and no clause
// for an empty spec. It returns an error that wraps [ErrSpecifier] for a clause without an
// operator, with a version that does not parse, with a wildcard after an operator other than ==
// and != or after a version that is no release, and for ~= with a version of fewer than two
// numbers.
func parseSpecifier(spec string) ([]clause, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	var out []clause
	offset := 0
	for part := range strings.SplitSeq(spec, clauseSeparator) {
		c, err := parseClause(part)
		if err != nil {
			return nil, fmt.Errorf("%w: %q: %w", ErrSpecifier, spec, err)
		}
		c.start = offset + len(part) - len(strings.TrimLeft(part, " \t"))
		c.end = c.start + len(c.raw)
		offset += len(part) + len(clauseSeparator)
		out = append(out, c)
	}
	return out, nil
}

// parseClause returns the clause that part of a specifier states. It returns the errors that
// [parseSpecifier] states for a clause.
func parseClause(part string) (clause, error) {
	raw := strings.TrimSpace(part)
	c := clause{raw: raw}
	for _, op := range operators {
		if rest, ok := strings.CutPrefix(raw, op); ok {
			c.op, c.text = op, strings.TrimSpace(rest)
			break
		}
	}
	if c.op == "" {
		return clause{}, fmt.Errorf("the clause %q, which has no operator", raw)
	}
	if c.op == opArbitrary {
		return c, nil
	}
	c.text, c.prefix = strings.CutSuffix(c.text, wildcard)
	v, err := pyworkspace.ParseVersion(c.text)
	if err != nil {
		return clause{}, fmt.Errorf("the clause %q: %w", raw, err)
	}
	c.version = v
	if c.prefix && ((c.op != opEqual && c.op != opNotEqual) || v.Pre != "" || v.HasPost || v.HasDev || v.Local != "") {
		return clause{}, fmt.Errorf("the clause %q, whose wildcard follows no release of == or !=", raw)
	}
	if c.op == opCompatible && len(v.Release) < 2 {
		return clause{}, fmt.Errorf("the clause %q, whose version has fewer than two numbers", raw)
	}
	return c, nil
}

// admits reports whether c admits v, by the rules of PEP 440. A pre-release counts as any other
// version, as a workspace of uv provides each member at its own version.
func (c *clause) admits(v *pyworkspace.Version) bool {
	public := *v
	public.Local = ""
	base, own := v.Base(), c.version.Base()
	sameBase := base.Compare(&own) == 0
	switch c.op {
	case opArbitrary:
		return strings.EqualFold(c.text, v.String())
	case opEqual:
		return c.equal(v)
	case opNotEqual:
		return !c.equal(v)
	case opLessEqual:
		return public.Compare(&c.version) <= 0
	case opGreaterEqual:
		return public.Compare(&c.version) >= 0
	case opLess:
		return v.Compare(&c.version) < 0 && (!v.IsPre() || c.version.IsPre() || !sameBase)
	case opGreater:
		return v.Compare(&c.version) > 0 && (!v.HasPost || c.version.HasPost || !sameBase) &&
			(v.Local == "" || !sameBase)
	}
	release := c.version.Release[:len(c.version.Release)-1]
	head := clause{version: pyworkspace.Version{Epoch: c.version.Epoch, Release: release}, prefix: true}
	return public.Compare(&c.version) >= 0 && head.equal(v)
}

// equal reports whether v is the version of c, a clause of == or !=: whether its release starts
// with the numbers of c for a clause with a wildcard, with a 0 for each number that v lacks, and
// whether v is the version of c otherwise. A clause without a local label compares v without one.
func (c *clause) equal(v *pyworkspace.Version) bool {
	if c.prefix {
		if v.Epoch != c.version.Epoch {
			return false
		}
		for i, n := range c.version.Release {
			if i >= len(v.Release) && n != 0 || i < len(v.Release) && v.Release[i] != n {
				return false
			}
		}
		return true
	}
	w := *v
	if c.version.Local == "" {
		w.Local = ""
	}
	return w.Compare(&c.version) == 0
}

// moved returns c moved to admit target with the same operator: the clauses >=, == and === name
// target, a clause with a wildcard and ~= name the numbers of target that they had, < moves its
// bound to the next value of its last number above target, and <= names target. A clause of <, <=,
// > and != that admits target stays as raw. It returns an error that wraps [ErrSpecifier] for a
// clause of > or != that excludes target.
func (c *clause) moved(target *pyworkspace.Version) (string, error) {
	switch {
	case c.op == opGreaterEqual || c.op == opArbitrary || (c.op == opEqual && !c.prefix):
		return c.op + target.String(), nil
	case c.op == opEqual:
		return opEqual + numbers(target, len(c.version.Release)) + wildcard, nil
	case c.op == opCompatible && target.Pre != "":
		return opCompatible + target.String(), nil
	case c.op == opCompatible:
		return opCompatible + numbers(target, len(c.version.Release)), nil
	case c.admits(target):
		return c.raw, nil
	case c.op == opLessEqual:
		return opLessEqual + target.String(), nil
	case c.op == opLess:
		bound := pyworkspace.Version{Release: make([]uint64, len(c.version.Release))}
		copy(bound.Release, target.Release)
		bound.Release[len(bound.Release)-1]++
		return opLess + bound.String(), nil
	}
	return "", fmt.Errorf(
		"%w: the clause %q, which excludes %s with its operator %s",
		ErrSpecifier,
		c.raw,
		target,
		c.op,
	)
}

// numbers returns the first n numbers of the release of v, with a 0 for each number that v lacks,
// separated by dots.
func numbers(v *pyworkspace.Version, n int) string {
	var b strings.Builder
	for i := range n {
		if i > 0 {
			b.WriteString(numberSeparator)
		}
		if i < len(v.Release) {
			b.WriteString(strconv.FormatUint(v.Release[i], 10))
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}
