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
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.yaml.in/yaml/v3"
)

// Toolchain is the name of the js toolchain in configuration and in reports. JavaScript and
// TypeScript name it.
const Toolchain workspace.Toolchain = "js"

// PNPMWorkspaceFile is the file of a pnpm workspace, at the root of the repository.
const PNPMWorkspaceFile = "pnpm-workspace.yaml"

// rootDir is the directory of the root of the repository, relative to the root.
const rootDir = "."

// nodeModules is the directory of the installed packages. npm and pnpm do not look for a package of
// a workspace below it.
const nodeModules = "node_modules"

// exclusion is the prefix of a glob of a workspace that excludes the directories that it matches.
const exclusion = "!"

// versionPrefixes are the characters that npm drops from the start of a version.
const versionPrefixes = "=v"

// pnpmWorkspace is the content of pnpm-workspace.yaml that a release reads.
type pnpmWorkspace struct {
	// Packages are the globs of the packages of the workspace.
	Packages []string `yaml:"packages"`
}

// member is a package of a workspace: its directory and its manifest.
type member struct {
	// manifest is the content of its package.json.
	manifest *Manifest

	// dir is the directory of the package, relative to the root of the repository and
	// slash-separated.
	dir string
}

// Discover returns a package of [Toolchain] for each package of the repository at root, in the
// order of their directories. The packages of a workspace are the directories with a package.json
// that a glob of the workspace matches, other than the root and a directory below node_modules. The
// globs are the packages of pnpm-workspace.yaml, or else the workspaces of the package.json at the
// root. A glob with an odd number of ! before it excludes the directories that it matches. The root
// of a workspace is no package. A repository without globs has one package, the package.json at its
// root, when that file has a name. A package.json without a name, such as one for the tools of a
// repository, is no package.
//
// A package has the name and the version of its package.json, and its requirements on the other
// packages as its requirements, with the requirement of the file. A shorthand of the workspace
// protocol becomes the requirement that pnpm packs for it, such as workspace:1.2.0 for workspace:*
// on a package at 1.2.0. npm drops a = and a v from the start of a version, and so does Discover. A
// private package without a version has the zero version.
//
// It returns no package and no error for a repository without a package.json at its root. It
// returns an error that wraps [ErrManifest] for a package.json or a pnpm-workspace.yaml that a
// release cannot read, a package of a workspace without a name, a package without a version that
// is not private, and a version that Semantic Versioning does not state. It also returns the error
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
	globs, err := pnpmGlobs(r)
	if err != nil {
		return nil, err
	}
	if len(globs) == 0 {
		globs = top.Workspaces
	}
	var members []member
	if len(globs) == 0 && top.Name != "" {
		members = append(members, member{dir: rootDir, manifest: top})
	}
	dirs, err := match(r.FS(), globs)
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
		if m.Name == "" {
			return nil, fmt.Errorf("%w: %s, whose package has no name", ErrManifest, path.Join(dir, ManifestFile))
		}
		members = append(members, member{dir: dir, manifest: m})
	}
	versions := make(map[string]version.Version, len(members))
	for i := range members {
		v, err := packageVersion(&members[i])
		if err != nil {
			return nil, err
		}
		versions[members[i].manifest.Name] = v
	}
	pkgs := make([]workspace.Package, 0, len(members))
	for _, p := range members {
		name := p.manifest.Name
		pkg := workspace.Package{
			Name: name, Toolchain: Toolchain, Dir: p.dir, Version: versions[name], Private: p.manifest.Private,
		}
		for _, e := range p.manifest.Requirements {
			if v, ok := versions[e.Name]; ok && e.Name != name {
				pkg.Deps = append(pkg.Deps, workspace.Dependency{Name: e.Name, Kind: e.Kind, Req: expand(e.Text, v)})
			}
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// read returns the manifest of the package.json in the directory dir of the repository of r, and
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

// pnpmGlobs returns the globs of the packages of the pnpm-workspace.yaml of the repository of r,
// and none for a repository without the file. It returns the error of reading the file, and an
// error that wraps [ErrManifest] for a file that does not parse.
func pnpmGlobs(r *os.Root) ([]string, error) {
	data, err := r.ReadFile(PNPMWorkspaceFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("workspace: read %s: %w", PNPMWorkspaceFile, err)
	}
	var w pnpmWorkspace
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrManifest, PNPMWorkspaceFile, err)
	}
	return w.Packages, nil
}

// match returns the directories of fsys, the repository, that a glob of globs matches and no
// exclusion of globs matches, in the order of their paths. It leaves out the root and the
// directories below node_modules. It drops a ./ and a / from the start of a glob, as npm does. It
// returns the error of a glob that does not parse, and of reading the repository.
func match(fsys fs.FS, globs []string) ([]string, error) {
	var include, exclude []string
	for _, glob := range globs {
		pattern := strings.TrimLeft(glob, exclusion)
		clean := strings.TrimLeft(path.Clean(pattern), "/")
		if (len(glob)-len(pattern))%2 == 1 {
			exclude = append(exclude, clean)
		} else {
			include = append(include, clean)
		}
	}
	var dirs []string
	for _, glob := range include {
		err := doublestar.GlobWalk(fsys, glob, func(dir string, d fs.DirEntry) error {
			if !d.IsDir() || dir == rootDir || slices.Contains(dirs, dir) {
				return nil
			}
			if slices.Contains(strings.Split(dir, "/"), nodeModules) {
				return doublestar.SkipDir
			}
			for _, x := range exclude {
				excluded, err := doublestar.Match(x, dir)
				if err != nil {
					return fmt.Errorf("the exclusion %q: %w", x, err)
				}
				if excluded {
					return nil
				}
			}
			dirs = append(dirs, dir)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("workspace: the glob %q of the workspace: %w", glob, err)
		}
	}
	slices.Sort(dirs)
	return dirs, nil
}

// packageVersion returns the version of the package p, as [Discover] states it. It returns an
// error that wraps [ErrManifest] for a package without a version that is not private, and for a
// version that Semantic Versioning does not state.
func packageVersion(p *member) (version.Version, error) {
	file := path.Join(p.dir, ManifestFile)
	text := strings.TrimLeft(strings.TrimSpace(p.manifest.Version), versionPrefixes)
	if text == "" {
		if p.manifest.Private {
			return version.Version{}, nil
		}
		return version.Version{}, fmt.Errorf("%w: %s, whose package has no version", ErrManifest, file)
	}
	v, err := version.Parse(text)
	if err != nil {
		return version.Version{}, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	return v, nil
}
