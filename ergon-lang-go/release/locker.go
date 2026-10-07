// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"maps"
	"path"
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	goworkspace "go.dokimi.dev/ergon/lang/go/workspace"
	"golang.org/x/mod/sumdb/dirhash"
)

// moduleHashes are the hashes that a go.sum records for the content of one module version.
type moduleHashes struct {
	// zip is the h1 hash of the zip of the version.
	zip string

	// mod is the h1 hash of the go.mod of the version.
	mod string
}

// Locker is the [language.Locker] of the toolchain go: the hashes that the go.sum of each module of
// the repository records for the versions of the other modules of the repository.
//
// # Concurrency
//
// A Locker is safe for concurrent use when its Snapshot is. Two calls of Lock must not write one
// repository at once, and Stale reads a repository that no Lock writes.
type Locker struct {
	// Snapshot returns the tree of the working tree of a directory as git would commit it, which
	// Stale and Lock build the zips of the modules from. It must not be nil.
	Snapshot func(ctx context.Context, dir string) (string, error)
}

var _ language.Locker = Locker{}

// Stale returns the go.sum of each module of the repository at root that records, for a module of
// pkgs at its version, a hash other than the hash of the module in the working tree: the h1 hash of
// the zip that golang.org/x/mod/zip writes from a snapshot of the working tree, or the h1 hash of
// its go.mod. Stale compares the lines that a go.sum has, as the go command checks them, so the
// go.sum of a module that records the hash of a go.mod alone is stale only when that hash differs.
// A module that replaces a module of pkgs with a directory reads that module from disk, so Stale
// skips its lines of that module. The paths are relative to root, slash-separated and sorted.
//
// It returns an error for a module of pkgs that the repository does not have, and the error of
// [goworkspace.Modules], of reading a go.sum, of the snapshot and of the zip.
func (l Locker) Stale(ctx context.Context, root string, pkgs []workspace.Package) ([]string, error) {
	r, err := l.run(root, pkgs)
	if err != nil {
		return nil, err
	}
	known := map[string]moduleHashes{}
	var stale []string
	for _, name := range slices.Sorted(maps.Keys(r.modules)) {
		m := r.modules[name]
		sums, err := readSums(r.root, m)
		if err != nil {
			return nil, err
		}
		pending := r.pending(m)
		for _, s := range sums {
			if !pending(s) {
				continue
			}
			h, ok := known[s.version.Path]
			if !ok {
				if h, err = r.hash(ctx, r.modules[s.version.Path]); err != nil {
					return nil, err
				}
				known[s.version.Path] = h
			}
			if (s.mod && s.hash != h.mod) || (!s.mod && s.hash != h.zip) {
				stale = append(stale, path.Join(m.Dir, goworkspace.SumFile))
				break
			}
		}
	}
	slices.Sort(stale)
	return stale, nil
}

// Lock rewrites the hashes that the go.sum of each module of the repository at root records for
// the modules of pkgs at their versions, other than in a module that replaces such a module with a
// directory. It removes those lines and runs go mod tidy in each such module against a module proxy
// that serves the modules of pkgs from a snapshot of the working tree, a module after every module
// of pkgs that it requires, as [Versioner.Apply] runs it. A go.sum that records the hashes of the
// working tree comes out unchanged, and a go.mod changes only where the new content of a module
// changes its requirements. Lock returns the go.mod and go.sum files whose content it changed,
// relative to root and slash-separated, in the order of the first change.
//
// It returns an error that wraps [ErrCycle] for modules that require each other without a directory
// replace, an error for a module of pkgs that the repository does not have, and the error of
// [goworkspace.Modules], of reading and of writing a go.sum, of git and of the go command with its
// output. On an error it also returns every go.mod and go.sum that it began to change.
func (l Locker) Lock(ctx context.Context, root string, pkgs []workspace.Package) ([]string, error) {
	r, err := l.run(root, pkgs)
	if err != nil {
		return nil, err
	}
	var tidy []*goworkspace.Module
	for _, name := range slices.Sorted(maps.Keys(r.modules)) {
		m := r.modules[name]
		sums, err := readSums(r.root, m)
		if err != nil {
			return r.files.touched, err
		}
		kept := slices.DeleteFunc(slices.Clone(sums), r.pending(m))
		if len(kept) == len(sums) {
			continue
		}
		if err := r.files.write(path.Join(m.Dir, goworkspace.SumFile), formatSums(kept)); err != nil {
			return r.files.touched, err
		}
		tidy = append(tidy, m)
	}
	if err := r.tidyAll(ctx, tidy); err != nil {
		return r.files.touched, err
	}
	return r.files.changed(), nil
}

// run returns the run of Stale or Lock in the repository at root, which releases each module of
// pkgs at its version. It returns the error of [newRun], and an error for a module of pkgs that the
// repository does not have.
func (l Locker) run(root string, pkgs []workspace.Package) (*run, error) {
	r, err := newRun(l.Snapshot, root)
	if err != nil {
		return nil, err
	}
	for k := range pkgs {
		if err := r.checkModule(pkgs[k].Name); err != nil {
			return nil, err
		}
		r.released[pkgs[k].Name] = "v" + pkgs[k].Version.String()
	}
	return r, nil
}

// pending returns the function that reports whether a line of the go.sum of m records a released
// module at its version, other than a module that m replaces with a directory.
func (r *run) pending(m *goworkspace.Module) func(sum) bool {
	return func(s sum) bool {
		return r.released[s.version.Path] == s.version.Version && !replaced(m.File, s.version.Path)
	}
}

// hash returns the hashes of the released module s at its version in the working tree: the h1
// hash of its zip, which [run.zip] writes, and the h1 hash of the go.mod in the zip, or of the go.mod
// that the go command synthesizes for a zip without one. It returns the error of zip.
func (r *run) hash(ctx context.Context, s *goworkspace.Module) (moduleHashes, error) {
	m, data, err := r.zip(ctx, s)
	if err != nil {
		return moduleHashes{}, err
	}
	// golang.org/x/mod/zip wrote data, so it reads, and every file that it names opens, without a
	// newline in its name.
	z, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	names := make([]string, len(z.File))
	for k, f := range z.File {
		names[k] = f.Name
	}
	zipHash, _ := dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return z.Open(name) })
	mod := modOf(m, data)
	modHash, _ := dirhash.Hash1([]string{goworkspace.ModFile}, func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(mod)), nil
	})
	return moduleHashes{zip: zipHash, mod: modHash}, nil
}
