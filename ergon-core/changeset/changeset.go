// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package changeset

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/version"
)

// Dir is the directory of the changeset files, relative to the root of the repository.
const Dir = ".changeset"

// Ext is the extension of a changeset file. README.md in [Dir] has it too, and is no changeset.
const Ext = ".md"

// Readme is the file of [Dir] that has the extension of a changeset and is none.
const Readme = "README.md"

// Changelog is the name of the changelog of a package, in the directory of the package, into which
// a release writes the summaries of its changesets.
const Changelog = "CHANGELOG.md"

// fence is the line that opens and closes the front matter.
const fence = "---"

// ErrInvalid is the error of [Parse] for a file that is no changeset.
var ErrInvalid = errors.New("changeset: invalid changeset")

// Release is one package that a changeset releases. Its JSON form is the one of changesets:
// {"name": ..., "type": ...}.
type Release struct {
	// Name is the name of the package as the changeset spells it: the name of the package's
	// registry, or <toolchain>:<name> for a name that two toolchains share.
	Name string `json:"name"`

	// Bump is the level of the release.
	Bump version.Bump `json:"type"`

	// Line is the line of the file that names the package, counted from 1, and 0 for a release
	// that no file states. [Format] and the JSON form do not write it.
	Line int `json:"-"`
}

// Changeset is one changeset file. The zero value is a changeset without an ID that releases
// nothing. Its JSON form is the one of changesets: {"id": ..., "summary": ..., "releases": [...]}.
type Changeset struct {
	// ID is the name of the file without its extension, such as add-the-unit-type-3f2a.
	ID string `json:"id"`

	// Summary is the body of the file without the blank lines and the spaces around it: the
	// entry of the changelog of each package that the changeset releases.
	Summary string `json:"summary"`

	// Releases are the packages of the front matter, in its order.
	Releases []Release `json:"releases"`
}

// Parse returns the changeset in data, the content of the file whose ID is id. The front matter
// opens on the first line and closes on the next line ---, and may be empty. A line of the front
// matter is empty, or a name, a colon and a level: the name in double quotes, in single quotes or
// unquoted, and the level one of none, patch, minor and major. An unquoted name ends at the last
// colon of its line. Each release records its line.
//
// It returns an error that wraps [ErrInvalid] and names the file and the line for a file whose
// first line is not ---, whose front matter does not close, with a line of the front matter that
// is no name and level, and with a name that the front matter names twice.
func Parse(id string, data []byte) (Changeset, error) {
	file := id + Ext
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) != fence {
		return Changeset{}, fmt.Errorf("%w: %s:1: the file does not open with the line %s", ErrInvalid, file, fence)
	}
	c := Changeset{ID: id}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == fence {
			c.Summary = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			return c, nil
		}
		if line == "" {
			continue
		}
		r, err := release(line)
		if err != nil {
			return Changeset{}, fmt.Errorf("%w: %s:%d: %w", ErrInvalid, file, i+1, err)
		}
		r.Line = i + 1
		if slices.ContainsFunc(c.Releases, func(other Release) bool { return other.Name == r.Name }) {
			return Changeset{}, fmt.Errorf("%w: %s:%d: the package %q, which the front matter names twice",
				ErrInvalid, file, i+1, r.Name)
		}
		c.Releases = append(c.Releases, r)
	}
	return Changeset{}, fmt.Errorf("%w: %s:%d: the front matter does not close with the line %s", ErrInvalid, file,
		len(lines), fence)
}

// Format returns c as changesets writes a changeset file: the front matter with each name in
// double quotes, a blank line, the summary and a newline. It writes no front matter line for a c
// without releases.
func Format(c *Changeset) []byte {
	var b bytes.Buffer
	b.WriteString(fence + "\n")
	for _, r := range c.Releases {
		fmt.Fprintf(&b, "%q: %s\n", r.Name, r.Bump)
	}
	b.WriteString(fence + "\n\n")
	b.WriteString(c.Summary)
	b.WriteString("\n")
	return b.Bytes()
}

// release returns the release that line states. It returns an error for a line without a colon,
// with an empty name, or with a level that is none of the four.
func release(line string) (Release, error) {
	before, after, ok := strings.CutLast(line, ":")
	if !ok {
		return Release{}, fmt.Errorf("the line %q, which is no name and level", line)
	}
	name := strings.TrimSpace(before)
	if len(name) >= 2 && (name[0] == '"' || name[0] == '\'') && name[len(name)-1] == name[0] {
		name = name[1 : len(name)-1]
	}
	if name == "" {
		return Release{}, fmt.Errorf("the line %q, which names no package", line)
	}
	bump, err := version.ParseBump(strings.TrimSpace(after))
	if err != nil {
		return Release{}, err
	}
	return Release{Name: name, Bump: bump}, nil
}
