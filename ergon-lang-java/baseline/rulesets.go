// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/option"
)

// rulesetSegment matches a segment of a rule set: a run of letters, digits and the characters . _
// and -. [Rulesets.Validate] also refuses the runs . and ..
var rulesetSegment = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Rulesets are the rule sets of PMD that lint-java checks the Java sources with. A rule set is the
// path of a rule set file of the repository, relative to its root, such as config/pmd.xml, or a
// rule set of PMD, such as its base ruleset rulesets/java/quickstart.xml. The init script of
// lint-java passes each path that is a file of the repository to PMD as a rule set file, so that a
// change of the file runs PMD again. PMD also looks for a file before it looks for a rule set of
// its own.
type Rulesets []string

// Validate returns an error that wraps [option.ErrInvalid] when r is empty, and when r has one of
// these rule sets:
//
//   - a rule set that r has twice
//   - a rule set with an empty segment, or with the segment . or ..
//   - a rule set with a character other than a letter, a digit or one of . _ - and /
//
// The init script writes each rule set into a string of Kotlin, and these characters do not need
// an escape there.
func (r Rulesets) Validate() error {
	if len(r) == 0 {
		return fmt.Errorf("%w: rulesets is empty", option.ErrInvalid)
	}
	for i, set := range r {
		if slices.Contains(r[:i], set) {
			return fmt.Errorf("%w: rule set %q, which appears twice", option.ErrInvalid, set)
		}
		for s := range strings.SplitSeq(set, "/") {
			if !rulesetSegment.MatchString(s) || s == "." || s == ".." {
				return fmt.Errorf("%w: rule set %q, which is not a relative path of segments of letters, digits, "+
					". _ and -", option.ErrInvalid, set)
			}
		}
	}
	return nil
}
