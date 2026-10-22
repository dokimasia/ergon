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
	rustworkspace "go.dokimi.dev/ergon/lang/rust/workspace"
)

// LockFile is the lockfile of a Cargo workspace, at the root of the repository.
const LockFile = "Cargo.lock"

// The program of a release, and the command and the flag of cargo that refresh the lockfile.
const (
	cargoProgram  = "cargo"
	updateCommand = "update"
	workspaceFlag = "--workspace"
)

// The delimiters of the strings of TOML of more than one line.
const (
	multiBasic   = `"""`
	multiLiteral = `'''`
)

// manifestMode is the mode of a Cargo.toml that a write would create. Apply writes only files that
// exist, whose modes a write keeps.
const manifestMode = 0o644

// Versioner is the [language.Versioner] of the toolchain rust, for the crates of a Cargo workspace.
//
// # Concurrency
//
// A Versioner is safe for concurrent use when its Command is. Two calls of Apply must not write one
// repository at once.
type Versioner struct {
	// Command runs the program name of the PATH with args in dir, with env added to the environment
	// of the process. Apply runs cargo update through it. It must not be nil.
	Command func(ctx context.Context, dir, name string, args, env []string) error
}

var _ language.Versioner = Versioner{}

// Resolve returns how a consumer that resolves the version requirement req alone treats v: Cargo
// picks the newest version that a requirement admits, so Resolve returns
// [language.ResolutionSelected] for a v that req admits by the rules of the semver crate, and
// [language.ResolutionExcluded] otherwise. An empty req, the requirement of a dependency without a
// version, admits every version. It returns an error that wraps [ErrRequirement] for a req that is
// no version requirement of Cargo.
func (Versioner) Resolve(req string, v version.Version) (language.Resolution, error) {
	r, err := parseRequirement(req)
	if err != nil {
		return 0, err
	}
	if r.matches(v) {
		return language.ResolutionSelected, nil
	}
	return language.ResolutionExcluded, nil
}

// Rewrite returns the version requirement req moved to admit v with its operators. Each comparator
// keeps its place, and the requirement keeps its spaces:
//
//   - A comparator of ^, ~, =, >= or no operator moves to the numbers of v that it has, as
//     cargo-release moves it: ^1.2 moves to ^1.3 for 1.3.0. For a pre-release v it moves to the
//     whole version, because only a comparator of three numbers admits a pre-release.
//   - A comparator with a wildcard keeps it: 1.2.* moves to 1.3.*.
//   - Rewrite keeps a comparator of <, <= or > that admits v. A comparator of < moves its bound to
//     the next value of its last number above v, and a comparator of <= moves to the numbers of v.
//
// Rewrite returns an empty req and the wildcard of every version as they are. It returns an error
// that wraps [ErrRequirement] for a req that is no version requirement of Cargo, for a comparator
// of > that excludes v, and for a req whose operators cannot admit v, such as 1.* for a
// pre-release.
func (Versioner) Rewrite(req string, v version.Version) (string, error) {
	r, err := parseRequirement(req)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	last := 0
	for i := range r.comparators {
		c := &r.comparators[i]
		text, err := c.moved(req, v)
		if err != nil {
			return "", err
		}
		b.WriteString(req[last:c.start])
		b.WriteString(text)
		last = c.end
	}
	b.WriteString(req[last:])
	out := b.String()
	if moved, err := parseRequirement(out); err != nil || !moved.matches(v) {
		return "", fmt.Errorf("%w: %q, whose operators admit no requirement of %s", ErrRequirement, req, v)
	}
	return out, nil
}

// Validate returns nil, as Cargo releases every version of Semantic Versioning.
func (Versioner) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply writes the new version and the rewritten requirements of each of edits into the
// Cargo.toml files of the workspace, and then refreshes Cargo.lock with cargo update --workspace
// when the repository has one. It writes the version of a crate in place of package.version, or of
// workspace.package.version of the Cargo.toml at the root for a crate that inherits it. It writes
// each requirement in place of the version of its dependency, or of the dependency of
// workspace.dependencies that the crate inherits. It keeps every other byte of each file, and the
// quotes of each string that it rewrites. It writes each file once, after the edits of every crate.
//
// It returns the files whose content it changed, relative to root and slash-separated, in the order
// of their first edit and with Cargo.lock last. It returns an error that wraps [ErrEdit] for an
// edit of a manifest without the version or the requirements that the discovery of its crate found,
// and for edits that write two requirements in place of one inherited requirement. It also returns
// the errors of the file system, of [rustworkspace.ParseManifest], of [rustworkspace.Requirements]
// and of cargo update. On an error it also returns the files that it changed before the error, and
// Cargo.lock after an error of cargo update.
func (v Versioner) Apply(ctx context.Context, root string, edits []language.Edit) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("release: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	w := &writes{root: r, manifests: map[string]*manifest{}}
	top, err := w.load(rustworkspace.ManifestFile)
	if err != nil {
		return nil, err
	}
	for i := range edits {
		err = w.edit(top, &edits[i])
		if err != nil {
			return nil, err
		}
	}
	changed, err := w.write()
	if err != nil || len(changed) == 0 {
		return changed, err
	}
	before, err := r.ReadFile(LockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return changed, nil
	}
	if err != nil {
		return changed, fmt.Errorf("release: read %s: %w", LockFile, err)
	}
	if err = v.Command(ctx, root, cargoProgram, []string{updateCommand, workspaceFlag}, nil); err != nil {
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
	span rustworkspace.Span
}

// manifest is a Cargo.toml that an Apply rewrites.
type manifest struct {
	// m is the manifest of data.
	m *rustworkspace.Manifest

	// file is the path of the file, relative to the root of the repository and slash-separated.
	file string

	// data are the bytes of the file.
	data []byte

	// replacements are the replacements of the edits, in the order of the edits.
	replacements []replacement
}

// writes are the Cargo.toml files of a repository that an Apply reads and rewrites.
type writes struct {
	// root is the repository.
	root *os.Root

	// manifests are the files that the Apply read, by their paths.
	manifests map[string]*manifest

	// order are the paths of the files with replacements, in the order of their first replacement.
	order []string
}

// load returns the Cargo.toml file of the repository, which it reads once. file is relative to the
// root of the repository and slash-separated. It returns the error of reading the file and of
// [rustworkspace.ParseManifest].
func (w *writes) load(file string) (*manifest, error) {
	if f, ok := w.manifests[file]; ok {
		return f, nil
	}
	data, err := w.root.ReadFile(filepath.FromSlash(file))
	if err != nil {
		return nil, fmt.Errorf("release: read %s: %w", file, err)
	}
	m, err := rustworkspace.ParseManifest(file, data)
	if err != nil {
		return nil, err
	}
	f := &manifest{m: m, file: file, data: data}
	w.manifests[file] = f
	return f, nil
}

// edit adds the replacements of e to the files of w, as [Versioner.Apply] states them. top is the
// Cargo.toml at the root. It returns the errors that Apply states for one edit.
func (w *writes) edit(top *manifest, e *language.Edit) error {
	f, err := w.load(path.Join(e.Package.Dir, rustworkspace.ManifestFile))
	if err != nil {
		return err
	}
	if e.Version != e.Package.Version {
		target, span := f, f.m.VersionSpan
		switch {
		case f.m.InheritsVersion && top.m.WorkspaceVersion == "":
			return fmt.Errorf(
				"%w: %s inherits workspace.package.version, which %s does not have",
				ErrEdit,
				f.file,
				top.file,
			)
		case f.m.InheritsVersion:
			target, span = top, top.m.WorkspaceVersionSpan
		case f.m.Version == "":
			return fmt.Errorf("%w: %s, which has no package.version", ErrEdit, f.file)
		}
		err = w.replace(target, span, e.Version.String())
		if err != nil {
			return err
		}
	}
	reqs, err := rustworkspace.Requirements(top.m, f.m, e.Package.Dir)
	if err != nil {
		return err
	}
	deps := e.Package.Deps
	reqs = slices.DeleteFunc(reqs, func(r rustworkspace.Requirement) bool {
		return !slices.ContainsFunc(deps, func(d workspace.Dependency) bool { return d.Name == r.Name })
	})
	if len(reqs) != len(deps) {
		return fmt.Errorf("%w: %s, whose requirements on the crates of the workspace changed", ErrEdit, f.file)
	}
	next := 0
	for _, req := range e.Requirements {
		for next < len(deps) && (deps[next].Name != req.Name || deps[next].Kind != req.Kind) {
			next++
		}
		if next == len(deps) {
			return fmt.Errorf("%w: %s, which has no %s requirement on %s", ErrEdit, f.file, req.Kind, req.Name)
		}
		entry := reqs[next]
		next++
		if req.Req == entry.Req {
			continue
		}
		if entry.Span.End == 0 {
			return fmt.Errorf("%w: %s, whose dependency on %s has no version", ErrEdit, f.file, req.Name)
		}
		// Requirements states a requirement in the file of the crate, or in the file at the root for
		// an inherited dependency.
		target := f
		if entry.File == top.file {
			target = top
		}
		err = w.replace(target, entry.Span, req.Req)
		if err != nil {
			return err
		}
	}
	return nil
}

// replace adds the replacement of the string at span of f with the string text to f. It returns an
// error that wraps [ErrEdit] for a span that an earlier replacement replaces with another text.
func (w *writes) replace(f *manifest, span rustworkspace.Span, text string) error {
	// The span of a string of TOML starts with its delimiter, of one or three quotes.
	old := f.data[span.Start:span.End]
	n := 1
	if bytes.HasPrefix(old, []byte(multiBasic)) || bytes.HasPrefix(old, []byte(multiLiteral)) {
		n = len(multiBasic)
	}
	delimiter := old[:min(n, len(old))]
	next := slices.Concat(delimiter, []byte(text), delimiter)
	for _, r := range f.replacements {
		if r.span != span {
			continue
		}
		if !bytes.Equal(r.text, next) {
			return fmt.Errorf("%w: %s, whose string %s two edits replace with %s and %s", ErrEdit, f.file, old, r.text,
				next)
		}
		return nil
	}
	if len(f.replacements) == 0 {
		w.order = append(w.order, f.file)
	}
	f.replacements = append(f.replacements, replacement{text: next, span: span})
	return nil
}

// write writes each file of w whose replacements change its content, in the order of their first
// replacement, and returns their paths. It returns the error of a write, with the paths that it
// wrote before the error.
func (w *writes) write() ([]string, error) {
	var changed []string
	for _, file := range w.order {
		f := w.manifests[file]
		slices.SortFunc(f.replacements, func(a, b replacement) int { return a.span.Start - b.span.Start })
		var out bytes.Buffer
		last := 0
		for _, r := range f.replacements {
			out.Write(f.data[last:r.span.Start])
			out.Write(r.text)
			last = r.span.End
		}
		out.Write(f.data[last:])
		if bytes.Equal(out.Bytes(), f.data) {
			continue
		}
		// The file exists, so WriteFile keeps its mode and does not use manifestMode.
		if err := w.root.WriteFile(filepath.FromSlash(file), out.Bytes(), manifestMode); err != nil {
			return changed, fmt.Errorf("release: write %s: %w", file, err)
		}
		changed = append(changed, file)
	}
	return changed, nil
}
