// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
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
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

// The lockfiles of the package managers, at the root of the repository.
const (
	NPMLockFile  = "package-lock.json"
	PNPMLockFile = "pnpm-lock.yaml"
)

// The programs of the package managers.
const (
	npmProgram  = "npm"
	pnpmProgram = "pnpm"
)

// The prefixes of a requirement on a directory and on a link, which changesets leaves as it is.
const (
	fileProtocol = "file:"
	linkProtocol = "link:"
)

// The operators of a caret and of a tilde range.
const (
	opCaret = "^"
	opTilde = "~"
)

// manifestMode is the mode of a package.json that a write would create. Apply writes only files
// that exist, whose modes a write keeps.
const manifestMode = 0o644

// quote is the quote of a string of JSON.
const quote = `"`

// lockfiles are the lockfiles of the package managers, each with its program and the arguments that
// refresh it without an install, in the order in which [Versioner.Apply] looks for them.
var lockfiles = []struct {
	file    string
	program string
	args    []string
}{
	{file: PNPMLockFile, program: pnpmProgram, args: []string{"install", "--lockfile-only"}},
	{file: NPMLockFile, program: npmProgram, args: []string{"install", "--package-lock-only"}},
}

// rangeOperators are the operators at the start of a range that changesets keeps when it moves the
// range to a new version, in the order in which it tests them.
var rangeOperators = []string{opCaret, opTilde, opGreaterEqual, opLessEqual, opGreater}

// Versioner is the [language.Versioner] of the js toolchain, for the packages of an npm or a pnpm
// workspace.
//
// # Concurrency
//
// A Versioner is safe for concurrent use when its Command is. Two calls of Apply must not write one
// repository at once.
type Versioner struct {
	// Command runs the program name of the PATH of the process with args in dir, with env added to
	// the environment of the process. Apply refreshes the lockfile through it with the package
	// manager. It must not be nil.
	Command func(ctx context.Context, dir, name string, args, env []string) error
}

var _ language.Versioner = Versioner{}

// Resolve returns how a consumer that resolves the range req alone treats v. npm and pnpm install
// the newest version that a range admits, so Resolve returns [language.ResolutionSelected] for a v
// that req admits, as the satisfies of node-semver 7.8.5 tests it, and
// [language.ResolutionExcluded] otherwise. It reads the range after the workspace protocol. A req
// that is not a range, such as file:../a or latest, does not admit a version, as changesets counts
// it. Resolve returns no error.
func (Versioner) Resolve(req string, v version.Version) (language.Resolution, error) {
	r, ok := parseRange(strings.TrimPrefix(req, jsworkspace.Protocol))
	if ok && slices.ContainsFunc(r, func(set []comparator) bool { return testSet(set, v) }) {
		return language.ResolutionSelected, nil
	}
	return language.ResolutionExcluded, nil
}

// Rewrite returns the range req moved to admit v, as changesets moves the range of a dependency on
// a package that it releases. A range of one lower and one upper comparator, such as
// >=1.0.0 <2.0.0, keeps its upper bound while the bound admits v. Otherwise the bound becomes <w,
// where w is v released once more at the level of v: major for x.0.0, minor for x.y.0 and patch for
// any other v. Any other range becomes v after the operator that starts it, ^, ~, >=, <=, > or none,
// so ^1.2.0 moves to ^1.3.0. A range after the workspace protocol keeps the protocol.
//
// Rewrite returns req as it is where changesets leaves a requirement as it is:
//
//   - a requirement of file: or of link:
//   - workspace:*, workspace:^ and workspace:~
//   - a requirement that is not a range
//   - a range of any version, such as *, for a v without a pre-release
//
// It returns the error of [version.Version.Bump] for a w past [version.MaxComponent].
func (Versioner) Rewrite(req string, v version.Version) (string, error) {
	if strings.HasPrefix(req, fileProtocol) || strings.HasPrefix(req, linkProtocol) {
		return req, nil
	}
	rest, protocol := strings.CutPrefix(req, jsworkspace.Protocol)
	if protocol && (rest == anyVersion || rest == opCaret || rest == opTilde) {
		return req, nil
	}
	r, ok := parseRange(rest)
	if !ok || (v.Pre == "" && len(r) == 1 && len(r[0]) == 1 && r[0][0].any) {
		return req, nil
	}
	out, err := moved(rest, r, v)
	if err != nil {
		return "", err
	}
	if protocol {
		return jsworkspace.Protocol + out, nil
	}
	return out, nil
}

// Validate returns nil: npm and pnpm publish every version of Semantic Versioning.
func (Versioner) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply writes the new version and the rewritten requirements of each of edits into the
// package.json of its package, and then refreshes the lockfile of the repository with its package
// manager: pnpm install --lockfile-only for pnpm-lock.yaml, and npm install --package-lock-only for
// package-lock.json. A repository without either gets none. Apply writes each value in place of its
// string in the file, and keeps every other byte. It leaves a shorthand of the workspace protocol,
// which pnpm replaces with the new version when it packs the package.
//
// It returns the files whose content it changed, relative to root and slash-separated, in the
// order of edits and with the lockfile last. It returns an error that wraps [ErrEdit] for an edit
// of a manifest that lacks the version or the requirements that the discovery of its package found.
// It also returns the errors of the file system, of [jsworkspace.ParseManifest] and of the package
// manager, and an error for a requirement that is not UTF-8. On an error it also returns the files
// that it changed before the error, and the lockfile after an error of the package manager.
func (v Versioner) Apply(ctx context.Context, root string, edits []language.Edit) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("release: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	var changed []string
	for i := range edits {
		file, err := apply(r, &edits[i])
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
	for _, lock := range lockfiles {
		before, err := r.ReadFile(lock.file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return changed, fmt.Errorf("release: read %s: %w", lock.file, err)
		}
		if err = v.Command(ctx, root, lock.program, lock.args, nil); err != nil {
			return append(changed, lock.file), fmt.Errorf("release: refresh %s: %w", lock.file, err)
		}
		after, err := r.ReadFile(lock.file)
		if err != nil {
			return append(changed, lock.file), fmt.Errorf("release: read %s: %w", lock.file, err)
		}
		if !bytes.Equal(before, after) {
			changed = append(changed, lock.file)
		}
		return changed, nil
	}
	return changed, nil
}

// replacement is a range of bytes of a file and the bytes that replace it.
type replacement struct {
	// text replaces the bytes of span.
	text []byte

	// span is the range that text replaces.
	span jsworkspace.Span
}

// moved returns the range text, which r is, moved to admit v, as [Versioner.Rewrite] states it.
// It returns the error of [version.Version.Bump].
func moved(text string, r semverRange, v version.Version) (string, error) {
	if len(r) == 1 && len(r[0]) == 2 && strings.Contains(text, opGreater) && strings.Contains(text, opLess) {
		set := r[0]
		lower := slices.IndexFunc(set, func(c comparator) bool { return strings.HasPrefix(c.op, opGreater) })
		upper := slices.IndexFunc(set, func(c comparator) bool { return strings.HasPrefix(c.op, opLess) })
		if lower >= 0 && upper >= 0 {
			bound := set[upper].value()
			if !testSet(set[upper:upper+1], v) {
				level := version.BumpMajor
				switch {
				case v.Patch != 0:
					level = version.BumpPatch
				case v.Minor != 0:
					level = version.BumpMinor
				}
				w, err := v.Bump(level)
				if err != nil {
					return "", fmt.Errorf("release: move %q to %s: %w", text, v, err)
				}
				bound = opLess + w.String()
			}
			if lower == 0 {
				return opGreaterEqual + v.String() + " " + bound, nil
			}
			return bound + " " + opGreaterEqual + v.String(), nil
		}
	}
	for _, op := range rangeOperators {
		if strings.HasPrefix(text, op) {
			return op + v.String(), nil
		}
	}
	return v.String(), nil
}

// apply writes e into the package.json of its package in the repository of r, as
// [Versioner.Apply] states. It returns the path of the file when it writes it, and the errors that
// Apply states for one edit.
func apply(r *os.Root, e *language.Edit) (string, error) {
	file := path.Join(e.Package.Dir, jsworkspace.ManifestFile)
	native := filepath.FromSlash(file)
	data, err := r.ReadFile(native)
	if err != nil {
		return "", fmt.Errorf("release: read %s: %w", file, err)
	}
	m, err := jsworkspace.ParseManifest(file, data)
	if err != nil {
		return "", err
	}
	var edits []replacement
	if e.Version != e.Package.Version {
		if m.Version == "" {
			return "", fmt.Errorf("%w: %s, which has no version", ErrEdit, file)
		}
		// A version has no character that JSON escapes.
		edits = append(edits, replacement{span: m.VersionSpan, text: []byte(quote + e.Version.String() + quote)})
	}
	entries := requirementsOn(m, e.Package.Deps)
	if len(entries) != len(e.Package.Deps) {
		return "", fmt.Errorf("%w: %s, whose requirements on the packages of the repository changed", ErrEdit, file)
	}
	at := 0
	for _, req := range e.Requirements {
		for at < len(e.Package.Deps) && (e.Package.Deps[at].Name != req.Name || e.Package.Deps[at].Kind != req.Kind) {
			at++
		}
		if at == len(e.Package.Deps) {
			return "", fmt.Errorf("%w: %s, which has no %s requirement on %s", ErrEdit, file, req.Kind, req.Name)
		}
		en := entries[at]
		at++
		if jsworkspace.Shorthand(en.Text) {
			continue
		}
		text, err := jsontext.AppendQuote(nil, req.Req)
		if err != nil {
			return "", fmt.Errorf("release: %s: the requirement %q: %w", file, req.Req, err)
		}
		edits = append(edits, replacement{span: en.Span, text: text})
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

// requirementsOn returns the entries of m on the packages that deps name, in the order of m: the
// requirements that the discovery of the package found as deps.
func requirementsOn(m *jsworkspace.Manifest, deps []workspace.Dependency) []*jsworkspace.Entry {
	var out []*jsworkspace.Entry
	for i := range m.Requirements {
		en := &m.Requirements[i]
		if slices.ContainsFunc(deps, func(d workspace.Dependency) bool { return d.Name == en.Name }) {
			out = append(out, en)
		}
	}
	return out
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
