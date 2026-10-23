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

// The parts of the value of files.includes of biome.json.
const (
	// includeAll is the glob of every path. The force-ignore of .ergon and the exclusions follow it
	// in files.includes.
	includeAll = "**"

	// ergonIgnored is the force-ignore of the directory .ergon in files.includes. Biome's scanner
	// then does not read a file under .ergon. Without it, Biome reads the local biome.json as a
	// second root configuration and stops.
	ergonIgnored = "!!**/.ergon"

	// excluded is the prefix of an exclusion in files.includes.
	excluded = "!"
)

// The layout of the managed biome.json.
const (
	// keyIndent is the indent of the line of the key includes.
	keyIndent = "    "

	// itemIndent is the indent of an item of files.includes, each on a line of its own.
	itemIndent = keyIndent + "  "

	// biomeLineWidth is the line width of the formatter of the managed biome.json. The template
	// renders it from [Choices.LineWidth].
	biomeLineWidth = 88
)

// excludeSegment matches a segment that is ** or a run of letters, digits and the characters
// . _ @ + ~ - *. [Exclude.Validate] also refuses the runs . and .. and a run with ** in it.
var excludeSegment = regexp.MustCompile(`^(\*\*|[A-Za-z0-9._@+~*-]+)$`)

// Exclude is the list of globs of the paths that fmt-js and lint-js skip. Each glob is relative to
// the root of the repository. The glob vendor matches the directory vendor, and src/**/*.gen.ts
// matches the files below src that end in .gen.ts. The toolkit skips a matched directory with every
// file below it. A glob can have * inside a segment and ** only as a whole segment. Biome, oxlint and
// oxfmt then match the same paths.
type Exclude []string

// Validate returns an error that wraps [option.ErrInvalid] when e has one of these globs:
//
//   - a glob that e has twice
//   - a glob with an empty segment, or with the segment . or ..
//   - a glob with a character other than a letter, a digit or one of . _ @ + ~ - *
//   - a glob with ** beside other characters in a segment
//   - a glob with two ** segments in a row
func (e Exclude) Validate() error {
	for i, glob := range e {
		if slices.Contains(e[:i], glob) {
			return fmt.Errorf("%w: exclude %q, which is named twice", option.ErrInvalid, glob)
		}
		segments := strings.Split(glob, "/")
		for k, s := range segments {
			valid := excludeSegment.MatchString(s) && s != "." && s != ".." &&
				(s == includeAll || !strings.Contains(s, includeAll))
			if !valid || (k > 0 && s == includeAll && segments[k-1] == includeAll) {
				return fmt.Errorf("%w: exclude %q, which is not a glob of segments of letters, digits, "+
					". _ @ + ~ - and *, with ** as a whole segment", option.ErrInvalid, glob)
			}
		}
	}
	return nil
}

// includes returns the value of files.includes of biome.json for e. The value starts with the glob
// of every path and the force-ignore of .ergon, and each glob of e follows with the prefix !. Each
// item is on a line of its own, as every array of the managed biome.json is.
func (e Exclude) includes() string {
	items := make([]string, 0, 2+len(e))
	items = append(items, `"`+includeAll+`"`, `"`+ergonIgnored+`"`)
	for _, glob := range e {
		items = append(items, `"`+excluded+glob+`"`)
	}
	return "[\n" + itemIndent + strings.Join(items, ",\n"+itemIndent) + "\n" + keyIndent + "]"
}
