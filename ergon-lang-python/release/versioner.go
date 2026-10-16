// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// LockFile is the lockfile of a uv workspace, at the root of the repository.
const LockFile = "uv.lock"

// The tool of the section python that refreshes the lockfile, and its arguments.
const (
	uvTool      = "uv"
	lockCommand = "lock"
)

// The quotes of a string of TOML, and the quotes of a literal string of more than one line.
const (
	basicQuote     = '"'
	literalQuote   = '\''
	escape         = '\\'
	multilineQuote = "'''"
)

// manifestMode is the mode of a pyproject.toml that a write would create. Apply writes only files
// that exist, whose modes a write keeps.
const manifestMode = 0o644

// escapes are the escapes of a basic string of TOML for the characters that a string cannot state
// as they are.
var escapes = map[rune]string{
	'\b': `\b`, '\t': `\t`, '\n': `\n`, '\f': `\f`, '\r': `\r`, basicQuote: `\"`, escape: `\\`,
}

// Versioner is the [language.Versioner] of the toolchain python, for the projects of a uv
// workspace.
//
// # Concurrency
//
// A Versioner is safe for concurrent use when its Tools is. Two calls of Apply must not write one
// repository at once.
type Versioner struct {
	// Tools runs the tool of the section python of .ergon.yaml with args in dir, with env added to
	// the environment of the process, as ergon tool run does. Apply runs uv lock through it. It must
	// not be nil.
	Tools func(ctx context.Context, dir, tool string, args, env []string) error
}

var _ language.Versioner = Versioner{}

// Resolve returns how a consumer that resolves the version specifier req alone treats v: a
// specifier of PEP 440 admits the versions that pip and uv select, so Resolve returns
// [language.ResolutionSelected] for a v that every clause of req admits, and
// [language.ResolutionExcluded] otherwise. An empty req admits every version. Resolve counts a
// pre-release as any other version. It returns an error that wraps [ErrSpecifier] for a req that
// is no version specifier, and one that wraps [pyworkspace.ErrVersion] for a v that PEP 440 does
// not have.
func (Versioner) Resolve(req string, v version.Version) (language.Resolution, error) {
	clauses, err := parseSpecifier(req)
	if err != nil {
		return 0, err
	}
	target, err := pyworkspace.FromSemVer(v)
	if err != nil {
		return 0, err
	}
	for i := range clauses {
		if !clauses[i].admits(&target) {
			return language.ResolutionExcluded, nil
		}
	}
	return language.ResolutionSelected, nil
}

// Rewrite returns the version specifier req moved to admit v with its operators, and >=v for an
// empty req: >=1.2.0 moves to >=1.3.0, ==1.2.* to ==1.3.*, ~=1.2 to ~=1.3, and <2 to <3 for 2.0.0.
// Each clause keeps its place and the specifier keeps its spaces. It returns an error that wraps
// [ErrSpecifier] for a req that is no version specifier and for a clause of > or != that excludes
// v, and one that wraps [pyworkspace.ErrVersion] for a v that PEP 440 does not have.
func (Versioner) Rewrite(req string, v version.Version) (string, error) {
	clauses, err := parseSpecifier(req)
	if err != nil {
		return "", err
	}
	target, err := pyworkspace.FromSemVer(v)
	if err != nil {
		return "", err
	}
	if len(clauses) == 0 {
		return opGreaterEqual + target.String(), nil
	}
	var b strings.Builder
	last := 0
	for i := range clauses {
		moved, err := clauses[i].moved(&target)
		if err != nil {
			return "", fmt.Errorf("%w, in %q", err, req)
		}
		b.WriteString(req[last:clauses[i].start])
		b.WriteString(moved)
		last = clauses[i].end
	}
	b.WriteString(req[last:])
	return b.String(), nil
}

// Validate returns an error that wraps [pyworkspace.ErrVersion] for a v that PEP 440 does not
// have, such as a pre-release other than a.N, b.N and rc.N, as pyproject.toml could not state it.
// It returns nil for any other release.
func (Versioner) Validate(p *workspace.Package, v version.Version) error {
	if _, err := pyworkspace.FromSemVer(v); err != nil {
		return fmt.Errorf("release: %s at %s: %w", p.Name, v, err)
	}
	return nil
}

// Apply writes the new version and the rewritten requirements of each of edits into the
// pyproject.toml of its package, and then refreshes uv.lock with uv lock when the repository has
// one. It writes the version of PEP 440 in place of project.version, and writes nothing for a
// dynamic version. It writes each requirement in place of the specifier of its string, and keeps
// the name, the extras and the marker of the string. It keeps every other byte of the file, and
// the single quotes of a string that it rewrites.
//
// It returns the files whose content it changed, relative to root and slash-separated, in the
// order of edits and with uv.lock last. It returns an error that wraps [ErrEdit] for an edit of a
// manifest without the version or the requirements that the discovery of its package found, and
// the errors of the file system, of [pyworkspace.ParseManifest] and of uv lock. On an error it also
// returns the files that it changed before the error, and uv.lock after an error of uv lock.
func (v Versioner) Apply(ctx context.Context, root string, edits []language.Edit) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("release: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	var changed []string
	for i := range edits {
		var file string
		file, err = apply(r, &edits[i])
		if file != "" {
			changed = append(changed, file)
		}
		if err != nil {
			return changed, err
		}
	}
	if len(changed) == 0 {
		return nil, nil
	}
	before, err := r.ReadFile(LockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return changed, nil
	}
	if err != nil {
		return changed, fmt.Errorf("release: read %s: %w", LockFile, err)
	}
	if err = v.Tools(ctx, root, uvTool, []string{lockCommand}, nil); err != nil {
		return append(changed, LockFile), fmt.Errorf("release: refresh %s: %w", LockFile, err)
	}
	after, err := r.ReadFile(LockFile)
	if err != nil {
		return append(changed, LockFile), fmt.Errorf("release: read %s: %w", LockFile, err)
	}
	if !bytes.Equal(before, after) {
		changed = append(changed, LockFile)
	}
	return changed, nil
}

// replacement is a range of bytes of a file and the bytes that replace it.
type replacement struct {
	// text replaces the bytes of span.
	text []byte

	// span is the range that text replaces.
	span pyworkspace.Span
}

// apply writes e into the pyproject.toml of its package in the repository of r, as
// [Versioner.Apply] states. It returns the path of the file when it writes it, and the errors that
// Apply states for one edit.
func apply(r *os.Root, e *language.Edit) (string, error) {
	file := path.Join(e.Package.Dir, pyworkspace.ManifestFile)
	native := filepath.FromSlash(file)
	data, err := r.ReadFile(native)
	if err != nil {
		return "", fmt.Errorf("release: read %s: %w", file, err)
	}
	m, err := pyworkspace.ParseManifest(file, data)
	if err != nil {
		return "", err
	}
	var edits []replacement
	if e.Version != e.Package.Version && !m.Dynamic {
		if m.Version == "" {
			return "", fmt.Errorf("%w: %s, which has no project.version", ErrEdit, file)
		}
		// The planner validated e.Version, so it converts.
		next, _ := pyworkspace.FromSemVer(e.Version)
		old := data[m.VersionSpan.Start:m.VersionSpan.End]
		edits = append(edits, replacement{span: m.VersionSpan, text: literal(old, next.String())})
	}
	entries := requirementsOn(m, e.Package.Deps)
	if len(entries) != len(e.Package.Deps) {
		return "", fmt.Errorf("%w: %s, whose requirements on the packages of the repository changed", ErrEdit, file)
	}
	next := 0
	for _, req := range e.Requirements {
		for next < len(e.Package.Deps) && (e.Package.Deps[next].Name != req.Name || e.Package.Deps[next].Kind != req.Kind) {
			next++
		}
		if next == len(e.Package.Deps) {
			return "", fmt.Errorf("%w: %s, which has no %s requirement on %s", ErrEdit, file, req.Kind, req.Name)
		}
		en := entries[next]
		next++
		text := en.Text[:en.Start] + req.Req + en.Text[en.End:]
		edits = append(edits, replacement{span: en.Span, text: literal(data[en.Span.Start:en.Span.End], text)})
	}
	out := splice(data, edits)
	if bytes.Equal(out, data) {
		return "", nil
	}
	// The file exists, so WriteFile keeps its mode and does not use manifestMode.
	if err := r.WriteFile(native, out, manifestMode); err != nil {
		return file, fmt.Errorf("release: write %s: %w", file, err)
	}
	return file, nil
}

// requirementsOn returns the requirements of m on the packages that deps name, other than those of
// a URL, in the order of m: the requirements that the discovery of the package found as deps.
func requirementsOn(m *pyworkspace.Manifest, deps []workspace.Dependency) []*pyworkspace.Entry {
	var out []*pyworkspace.Entry
	for i := range m.Requirements {
		en := &m.Requirements[i]
		name := pyworkspace.Normalize(en.Name)
		if !en.URL && slices.ContainsFunc(deps, func(d workspace.Dependency) bool { return d.Name == name }) {
			out = append(out, en)
		}
	}
	return out
}

// literal returns text as a string of TOML in place of the literal old: in single quotes when old
// is a literal string of one line, and as a basic string with escapes otherwise. text is the text
// of old with a version or a specifier in place of another, so text has no single quote and no line
// break where old has none.
func literal(old []byte, text string) []byte {
	if len(old) > 0 && old[0] == literalQuote && !bytes.HasPrefix(old, []byte(multilineQuote)) {
		return []byte(string(literalQuote) + text + string(literalQuote))
	}
	var b bytes.Buffer
	b.WriteRune(basicQuote)
	for _, c := range text {
		switch e, ok := escapes[c]; {
		case ok:
			b.WriteString(e)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, `\u%04X`, c)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteRune(basicQuote)
	return b.Bytes()
}

// splice returns data with each replacement of edits in place of its span. The spans do not
// overlap.
func splice(data []byte, edits []replacement) []byte {
	slices.SortFunc(edits, func(a, b replacement) int { return a.span.Start - b.span.Start })
	var b bytes.Buffer
	last := 0
	for _, e := range edits {
		b.Write(data[last:e.span.Start])
		b.Write(e.text)
		last = e.span.End
	}
	b.Write(data[last:])
	return b.Bytes()
}
