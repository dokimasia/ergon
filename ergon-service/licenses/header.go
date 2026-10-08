// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/skywalking-eyes/pkg/comments"
	"github.com/apache/skywalking-eyes/pkg/header"
	lcs "github.com/apache/skywalking-eyes/pkg/license"
	"github.com/apache/skywalking-eyes/pkg/logger"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/sirupsen/logrus"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/service/vcs"
)

// Kind is what is wrong with the header of a file.
type Kind string

// The kinds of a finding.
const (
	// Missing is a file without a header.
	Missing Kind = "missing"

	// Outdated is a file whose header names another owner or another license. [Fix] replaces the
	// header and keeps its years.
	Outdated Kind = "outdated"

	// Conflict is a file whose header has a line that is neither a copyright notice, nor a tag of
	// SPDX, nor empty. [Fix] leaves the file.
	Conflict Kind = "conflict"

	// Unsupported is a file with a comment style whose content is not text, such as a file with a
	// null byte. Neither [Check] nor [Fix] reads its header.
	Unsupported Kind = "unsupported"
)

// Finding is a file whose header does not match the configuration.
type Finding struct {
	// Path is the path of the file in the repository, slash-separated.
	Path string `json:"path"`

	// Kind is what is wrong with the header.
	Kind Kind `json:"kind"`

	// Line is the number of the first line of the header that is no header text, counted from 1,
	// for a Conflict, and 0 for any other kind.
	Line int `json:"line,omitempty"`
}

// Report is what [Check] or [Fix] found in a repository.
type Report struct {
	// Findings are the files whose header does not match the configuration, sorted by path. Fix
	// reports the files that it leaves: the conflicts and the unsupported files.
	Findings []Finding `json:"findings"`

	// Fixed are the files that Fix wrote, sorted, and nil for Check.
	Fixed []string `json:"fixed,omitempty"`

	// Checked is the number of files that the configuration gives a header.
	Checked int `json:"checked"`

	// Skipped is the number of files without a header: the excluded files, the changesets and the
	// changelogs, the files that a tool generated or that ergon init manages, the files without a
	// comment style, and the paths that are no regular file, such as a link or a tracked file that
	// the working tree deleted.
	Skipped int `json:"skipped"`
}

// bom is the byte-order mark of UTF-8, which Fix keeps before the header.
var bom = []byte("\xef\xbb\xbf")

// generated matches the first line of a file that a tool writes: the marker of generated Go code,
// which other generators write too, and the comment of a file that ergon init manages.
var generated = regexp.MustCompile(`Code generated .* DO NOT EDIT|Managed by ergon init`)

// silence sets the logger of skywalking-eyes, a variable of its package logger that writes every
// message at the level debug to standard output, to discard every message. It runs once per
// process, before the first call into the library.
var silence = sync.OnceFunc(func() {
	logger.Log.SetOutput(io.Discard)
	logger.Log.SetLevel(logrus.PanicLevel)
})

// file is a file that the configuration gives a header.
type file struct {
	// style is the comment style of the header.
	style comments.CommentStyle

	// after is the compiled preamble of the style, or nil.
	after *regexp.Regexp

	// path is the path of the file in the repository, slash-separated.
	path string

	// name is the path of the file on the system.
	name string

	// content is the content of the file without its byte-order mark.
	content []byte

	// mode is the mode of the file.
	mode fs.FileMode

	// bom reports that the file starts with the byte-order mark of UTF-8.
	bom bool
}

// run is a check or a fix of the headers of a repository.
type run struct {
	// c is the configuration.
	c *Config

	// t is the table of the comment styles.
	t *table

	// pattern matches a normalized file whose header states the owner and the license of c, with
	// any year, list of years or range of years.
	pattern *regexp.Regexp

	// root is the directory of the repository.
	root string

	// content is the text of the header, with the owner and the year as the placeholders of the
	// library.
	content string

	// report is what the run found.
	report Report
}

// Check returns the report of the files of the working tree of root whose header does not match c:
// each file that git tracks, or would track, and that has a comment style, as [Config.Styles] and
// the table of skywalking-eyes resolve it. It skips a file that c excludes, a changeset, a
// changelog, a file that a tool generated or that ergon init manages, and a file without a comment
// style. A changeset is a Markdown file of .changeset other than its README.md: its front matter
// opens the file and its body is the entry of the changelogs, so a header has no place in it. A
// changelog is a file named CHANGELOG.md in any directory: ergon release version creates it
// without a header, and adds an entry to it for each release. A header matches when the file,
// normalized as the library normalizes a header, states Copyright <owner> <years> and
// SPDX-License-Identifier: <spdx>, with any year, list of years or range of years. Check writes
// nothing.
//
// It returns an error that wraps [vcs.ErrGit] for a root outside a working tree of git and for a
// ctx that ends before git lists the files, and the error of a file that does not read. ctx bounds
// git alone: the check of the files reads the local file system.
func Check(ctx context.Context, root string, c *Config) (Report, error) {
	r := start(root, c)
	err := r.walk(ctx, func(f *file) error {
		if kind, line := r.inspect(f); kind != "" {
			r.report.Findings = append(r.report.Findings, Finding{Path: f.path, Kind: kind, Line: line})
		}
		return nil
	})
	return r.report, err
}

// Fix adds the header of c to each file that [Check] reports missing, and replaces the header of
// each file that it reports outdated, in one write per file. A new header states year. A replaced
// header keeps its years, and states year when it states none. Fix keeps the preamble of the
// style, such as a shebang line, and the byte-order mark of UTF-8 before the header, and it keeps
// the mode of each file. It returns the report of the files that it wrote and of the files that it
// leaves: the conflicts and the unsupported files.
//
// It returns the errors of Check, and the error of a file that does not write. A file that Fix
// wrote before an error keeps its new header.
func Fix(ctx context.Context, root string, c *Config, year int) (Report, error) {
	r := start(root, c)
	err := r.walk(ctx, func(f *file) error {
		kind, line := r.inspect(f)
		if kind == "" {
			return nil
		}
		if kind == Conflict || kind == Unsupported {
			r.report.Findings = append(r.report.Findings, Finding{Path: f.path, Kind: kind, Line: line})
			return nil
		}
		if err := r.fix(f, year); err != nil {
			return err
		}
		r.report.Fixed = append(r.report.Fixed, f.path)
		return nil
	})
	return r.report, err
}

// start returns a run of c over the repository at root, with the logger of the library silenced.
func start(root string, c *Config) *run {
	silence()
	pattern := header.ConfigHeader{License: header.LicenseConfig{
		Pattern: `Copyright ` + regexp.QuoteMeta(c.Owner) + ` \d{4}(?:\s*[-,]\s*\d{4})*\s+SPDX-License-Identifier: ` +
			regexp.QuoteMeta(string(c.SPDX)) + `(?:\s|$)`,
	}}
	return &run{
		c:       c,
		t:       styles(),
		pattern: pattern.NormalizedPattern(),
		root:    root,
		content: "Copyright [owner] [year]\nSPDX-License-Identifier: " + string(c.SPDX),
	}
}

// walk calls visit for each file of the repository that the configuration gives a header, in the
// order of their paths, and counts the files that it checks and skips. It returns the error of git,
// and stops at the first error of a file or of visit, and returns it.
func (r *run) walk(ctx context.Context, visit func(*file) error) error {
	paths, err := vcs.Files(ctx, r.root)
	if err != nil {
		return err
	}
	for _, p := range paths {
		f, ok, err := r.open(p)
		if err != nil {
			return err
		}
		if !ok {
			r.report.Skipped++
			continue
		}
		r.report.Checked++
		if err := visit(f); err != nil {
			return err
		}
	}
	return nil
}

// open returns the file at p, a path of the repository, and reports whether the configuration
// gives it a header: a regular file that c does not exclude, that is no changeset and no changelog,
// that has a comment style, and whose first line has no marker of a generated or a managed file. A
// tracked file that the working tree deleted has none. It returns the error of a path that does
// not stat, other than a missing one, and of a file that does not read.
func (r *run) open(p string) (*file, bool, error) {
	dir, base := path.Split(p)
	excluded := (dir == changeset.Dir+"/" && strings.HasSuffix(base, changeset.Ext) &&
		!strings.EqualFold(base, changeset.Readme)) || base == changeset.Changelog ||
		slices.ContainsFunc(r.c.Exclude, func(glob string) bool {
			matched, _ := doublestar.Match(glob, p)
			return matched
		})
	if excluded {
		return nil, false, nil
	}
	style, ok := r.t.resolve(p, r.c.Styles)
	if !ok {
		return nil, false, nil
	}
	name := filepath.Join(r.root, filepath.FromSlash(p))
	info, err := os.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("licenses: stat %s: %w", p, err)
	}
	if !info.Mode().IsRegular() {
		return nil, false, nil
	}
	content, err := os.ReadFile(name)
	if err != nil {
		return nil, false, fmt.Errorf("licenses: read %s: %w", p, err)
	}
	content, marked := bytes.CutPrefix(content, bom)
	first, _, _ := bytes.Cut(content, []byte("\n"))
	if generated.Match(first) {
		return nil, false, nil
	}
	f := &file{
		style:   style,
		after:   r.t.after[style.ID],
		path:    p,
		name:    name,
		content: content,
		mode:    info.Mode(),
		bom:     marked,
	}
	return f, true, nil
}

// inspect returns what is wrong with the header of f, and the number of the line of a conflict, or
// the empty kind for a file whose header matches. A file that the content sniffer of net/http does
// not read as text, as the library skips it, is unsupported. A file whose header does not match is
// a conflict when its header has a line that is no header text, outdated when it has a header, and
// missing otherwise.
func (r *run) inspect(f *file) (Kind, int) {
	if !strings.HasPrefix(http.DetectContentType(f.content), "text/") {
		return Unsupported, 0
	}
	if r.pattern.MatchString(lcs.NormalizeHeader(string(f.content))) {
		return "", 0
	}
	b, ok := find(f.content, &f.style, f.after)
	if !ok {
		return Missing, 0
	}
	if b.conflict > 0 {
		return Conflict, b.conflict
	}
	return Outdated, 0
}

// fix writes the header of the run into f, which has no header or an outdated one, in one write. It
// removes an outdated header and keeps its years, renders the header in the style of f with the
// library, and inserts it after the preamble of the style, as the library inserts it. A new header
// states year. fix keeps the byte-order mark and the mode of f. It returns the error of the write.
func (r *run) fix(f *file, year int) error {
	stated := strconv.Itoa(year)
	content := f.content
	if b, ok := find(f.content, &f.style, f.after); ok {
		if b.years != "" {
			stated = b.years
		}
		content = slices.Concat(content[:b.start], content[b.end:])
	}
	cfg := header.ConfigHeader{License: header.LicenseConfig{
		Content:        r.content,
		CopyrightOwner: r.c.Owner,
		CopyrightYear:  stated,
	}}
	// Every style of the table has a start, which is all that GenerateLicenseHeader requires.
	text, _ := header.GenerateLicenseHeader(&f.style, &cfg)
	content = insert(content, text, &f.style, f.after)
	if f.bom {
		content = slices.Concat(bom, content)
	}
	if err := os.WriteFile(f.name, content, f.mode); err != nil {
		return fmt.Errorf("licenses: write %s: %w", f.path, err)
	}
	return nil
}
