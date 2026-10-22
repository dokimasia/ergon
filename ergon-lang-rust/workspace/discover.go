// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Toolchain is the name of the toolchain of Rust in configuration and in reports.
const Toolchain workspace.Toolchain = "rust"

// ManifestFile is the manifest of a crate, in the directory of the crate, and of a workspace, at
// its root.
const ManifestFile = "Cargo.toml"

// VersionSource is the Source of a crate whose version is workspace.package.version of the
// Cargo.toml at the root, so the crates that inherit it share one version.
const VersionSource = ManifestFile + " workspace.package.version"

// Discover returns a package of [Toolchain] for each crate of the Cargo workspace of the repository
// at root, in the order of their directories with the root first, as Cargo finds the members:
//
//   - the crate of the Cargo.toml at the root, when it has a table package
//   - the crate of each directory that a path of workspace.members matches, which may be a glob,
//     and that no path of workspace.exclude contains
//   - the crate of each path of a dependency of a member, when the root contains the path and no
//     path of workspace.exclude contains it
//
// A Cargo.toml at the root without a table workspace has one crate. A package has the name and the
// directory of its crate. Its version is package.version, or workspace.package.version of the root
// for a crate that inherits it, with [VersionSource] as its Source. A crate without a version has
// the zero version, and is private like a crate that [Manifest] reports private. The requirements
// of the package are the [Requirements] of its crate on the other crates of the workspace.
//
// It returns no package and no error for a repository without a Cargo.toml at its root, and for a
// Cargo.toml without a table package and a table workspace. It returns an error that wraps
// [ErrManifest] for a Cargo.toml that a release cannot read, a path of workspace.members that matches
// nothing, a member without a Cargo.toml or without a table package, a crate without a name, an
// inherited version or dependency that the root does not have, and a version that does not parse.
// It also returns the error of opening root, of reading a file, and of a glob that does not parse.
func Discover(_ context.Context, root string) ([]workspace.Package, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	top, ok, err := read(r, rootDir)
	if err != nil || !ok {
		return nil, err
	}
	crates := map[string]*Manifest{}
	if top.Package {
		crates[rootDir] = top
	}
	if top.Workspace {
		if err := members(r, top, crates); err != nil {
			return nil, err
		}
	}
	dirs := slices.Sorted(maps.Keys(crates))
	if top.Package {
		dirs = slices.Insert(slices.DeleteFunc(dirs, func(dir string) bool { return dir == rootDir }), 0, rootDir)
	}
	for _, dir := range dirs {
		if crates[dir].Name == "" {
			return nil, fmt.Errorf("%w: %s, whose crate has no name", ErrManifest, path.Join(dir, ManifestFile))
		}
	}
	pkgs := make([]workspace.Package, 0, len(dirs))
	for _, dir := range dirs {
		pkg, err := crate(top, crates, dir)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// members adds the members of the workspace of top in the repository of r to crates, by their
// directories, as [Discover] states them. It returns the errors that Discover states for the
// members.
func members(r *os.Root, top *Manifest, crates map[string]*Manifest) error {
	var queue []string
	for _, glob := range top.Members {
		matched := false
		err := doublestar.GlobWalk(r.FS(), path.Clean(glob), func(dir string, d fs.DirEntry) error {
			matched = true
			if d.IsDir() && dir != rootDir && !excluded(top, dir) && !slices.Contains(queue, dir) {
				queue = append(queue, dir)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("workspace: the member %q of workspace.members: %w", glob, err)
		}
		if !matched {
			return fmt.Errorf(
				"%w: %s: the member %q of workspace.members matches nothing",
				ErrManifest,
				ManifestFile,
				glob,
			)
		}
	}
	for _, dir := range queue {
		if err := load(r, crates, dir); err != nil {
			return err
		}
	}
	// The members of the queue and the crate at the root add the crates of their paths, and each
	// crate that joins the queue adds the crates of its own paths.
	if top.Package {
		queue = append([]string{rootDir}, queue...)
	}
	for i := 0; i < len(queue); i++ {
		reqs, err := Requirements(top, crates[queue[i]], queue[i])
		if err != nil {
			return err
		}
		for _, req := range reqs {
			if crates[req.Dir] != nil || !filepath.IsLocal(req.Dir) || excluded(top, req.Dir) {
				continue
			}
			if err := load(r, crates, req.Dir); err != nil {
				return err
			}
			queue = append(queue, req.Dir)
		}
	}
	return nil
}

// load adds the crate of the directory dir of the repository of r to crates. It returns an error
// that wraps [ErrManifest] for a directory without a Cargo.toml and for a Cargo.toml without a
// table package, and the errors of [read].
func load(r *os.Root, crates map[string]*Manifest, dir string) error {
	file := path.Join(dir, ManifestFile)
	m, ok, err := read(r, dir)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s, which a crate of the workspace needs, does not exist", ErrManifest, file)
	}
	if !m.Package {
		return fmt.Errorf("%w: %s, which a crate of the workspace needs, has no table package", ErrManifest, file)
	}
	crates[dir] = m
	return nil
}

// excluded reports whether a path of workspace.exclude of top is dir or contains it.
func excluded(top *Manifest, dir string) bool {
	return slices.ContainsFunc(top.Exclude, func(ex string) bool {
		ex = path.Clean(ex)
		return dir == ex || strings.HasPrefix(dir, ex+"/")
	})
}

// crate returns the package of the crate of the directory dir of crates, the crates of the
// workspace of top, as [Discover] states it. It returns an error that wraps [ErrManifest] for an
// inherited version that top does not have and for a version that does not parse, and the errors
// of [Requirements].
func crate(top *Manifest, crates map[string]*Manifest, dir string) (workspace.Package, error) {
	m := crates[dir]
	file := path.Join(dir, ManifestFile)
	pkg := workspace.Package{Name: m.Name, Toolchain: Toolchain, Dir: dir, Private: m.Private}
	text := m.Version
	if m.InheritsVersion {
		if top.WorkspaceVersion == "" {
			return workspace.Package{}, fmt.Errorf("%w: %s inherits workspace.package.version, which %s does not have",
				ErrManifest, file, ManifestFile)
		}
		text, file, pkg.Source = top.WorkspaceVersion, ManifestFile, VersionSource
	}
	if text == "" {
		// Cargo does not publish a crate without a version, whose version is 0.0.0.
		pkg.Private = true
	} else {
		v, err := version.Parse(text)
		if err != nil {
			return workspace.Package{}, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
		}
		pkg.Version = v
	}
	reqs, err := Requirements(top, m, dir)
	if err != nil {
		return workspace.Package{}, err
	}
	for _, req := range reqs {
		if dep := crates[req.Dir]; dep != nil && req.Dir != dir {
			pkg.Deps = append(pkg.Deps, workspace.Dependency{Name: dep.Name, Kind: req.Kind, Req: req.Req})
		}
	}
	return pkg, nil
}

// read returns the manifest of the Cargo.toml in the directory dir of the repository of r, and
// reports whether the directory has one. It returns the error of reading the file, and of
// [ParseManifest].
func read(r *os.Root, dir string) (*Manifest, bool, error) {
	file := path.Join(dir, ManifestFile)
	data, err := r.ReadFile(filepath.FromSlash(file))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("workspace: read %s: %w", file, err)
	}
	m, err := ParseManifest(file, data)
	return m, err == nil, err
}
