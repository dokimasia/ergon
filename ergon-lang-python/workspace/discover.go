// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Toolchain is the name of the toolchain of Python in configuration and in reports.
const Toolchain workspace.Toolchain = "python"

// ManifestFile is the manifest of a project, in the directory of the project.
const ManifestFile = "pyproject.toml"

// rootDir is the directory of the root of the repository, relative to the root.
const rootDir = "."

// project is a project of a workspace: its directory and its manifest.
type project struct {
	// manifest is the content of its pyproject.toml.
	manifest *Manifest

	// dir is the directory of the project, relative to the root of the repository and
	// slash-separated.
	dir string
}

// Discover returns a package of [Toolchain] for each project of the uv workspace of the repository
// at root, in the order of their directories with the root first. The projects are the project of
// the pyproject.toml at the root, and the project of each directory that a glob of the members of
// tool.uv.workspace matches and no glob of its exclude matches. A directory without a
// pyproject.toml is no project, as uv skips it.
//
// A package has the name of its project in the normal form of PEP 503, its directory, its version
// as Semantic Versioning states it, and the requirements of its project on the other projects as
// its requirements, with their specifiers. A requirement of a URL is no requirement of a package. A
// project with a dynamic version has the highest version of the headings of its CHANGELOG.md, and
// the zero version without one. A project is private when [Manifest] reports it private.
//
// It returns no package and no error for a repository without a pyproject.toml at its root, and for
// a pyproject.toml without a project and a workspace. It returns an error that wraps [ErrManifest]
// for a pyproject.toml that a release cannot read, a member without a project, a project without a
// name, and a project without a version that Semantic Versioning states. It also returns the error
// of opening root, of reading a file, and of a glob that does not parse.
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
	var projects []project
	if top.Project {
		projects = append(projects, project{dir: rootDir, manifest: top})
	}
	dirs, err := members(r.FS(), top)
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		m, ok, err := read(r, dir)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if !m.Project {
			return nil, fmt.Errorf("%w: %s, which is a member of the workspace without a table project", ErrManifest,
				path.Join(dir, ManifestFile))
		}
		projects = append(projects, project{dir: dir, manifest: m})
	}
	names := make(map[string]bool, len(projects))
	for _, p := range projects {
		if p.manifest.Name == "" {
			return nil, fmt.Errorf("%w: %s, whose project has no name", ErrManifest, path.Join(p.dir, ManifestFile))
		}
		names[Normalize(p.manifest.Name)] = true
	}
	pkgs := make([]workspace.Package, 0, len(projects))
	for _, p := range projects {
		v, err := projectVersion(r, &p)
		if err != nil {
			return nil, err
		}
		name := Normalize(p.manifest.Name)
		pkg := workspace.Package{Name: name, Toolchain: Toolchain, Dir: p.dir, Version: v, Private: p.manifest.Private}
		for _, e := range p.manifest.Requirements {
			if dep := Normalize(e.Name); !e.URL && dep != name && names[dep] {
				pkg.Deps = append(pkg.Deps, workspace.Dependency{Name: dep, Kind: e.Kind, Req: e.Specifier})
			}
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// read returns the manifest of the pyproject.toml in the directory dir of the repository of r, and
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

// members returns the directories of fsys, the repository, other than its root, that a glob of the
// members of the workspace of top matches and that no glob of its exclude matches, in the order of
// their paths. It returns none for a top without a workspace. It returns the error of a glob that
// does not parse, and of reading the repository.
func members(fsys fs.FS, top *Manifest) ([]string, error) {
	var dirs []string
	for _, glob := range top.Members {
		err := doublestar.GlobWalk(fsys, path.Clean(glob), func(dir string, d fs.DirEntry) error {
			if !d.IsDir() || dir == rootDir || slices.Contains(dirs, dir) {
				return nil
			}
			excluded, err := matchesAny(top.Exclude, dir)
			if err != nil || excluded {
				return err
			}
			dirs = append(dirs, dir)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("workspace: the member %q of tool.uv.workspace: %w", glob, err)
		}
	}
	slices.Sort(dirs)
	return dirs, nil
}

// matchesAny reports whether a glob of globs matches dir. It returns the error of a glob that does
// not parse.
func matchesAny(globs []string, dir string) (bool, error) {
	for _, glob := range globs {
		ok, err := doublestar.Match(path.Clean(glob), dir)
		if err != nil {
			return false, fmt.Errorf("workspace: the exclude %q of tool.uv.workspace: %w", glob, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// projectVersion returns the version of the project p of the repository of r, as [Discover] states
// it. It returns an error that wraps [ErrManifest] for a project without a version and for a
// version that Semantic Versioning does not state, and the error of reading the changelog.
func projectVersion(r *os.Root, p *project) (version.Version, error) {
	file := path.Join(p.dir, ManifestFile)
	if p.manifest.Dynamic {
		changelog := path.Join(p.dir, changeset.Changelog)
		data, err := r.ReadFile(filepath.FromSlash(changelog))
		if errors.Is(err, fs.ErrNotExist) {
			return version.Version{}, nil
		}
		if err != nil {
			return version.Version{}, fmt.Errorf("workspace: read %s: %w", changelog, err)
		}
		return changeset.Latest(data), nil
	}
	if p.manifest.Version == "" {
		return version.Version{}, fmt.Errorf("%w: %s, whose project has no version", ErrManifest, file)
	}
	v, err := ParseVersion(p.manifest.Version)
	if err != nil {
		return version.Version{}, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	s, err := v.SemVer()
	if err != nil {
		return version.Version{}, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	return s, nil
}
