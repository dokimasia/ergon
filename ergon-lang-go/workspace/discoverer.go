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
	"regexp"
	"strings"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Toolchain is the name of the toolchain of Go in configuration and in reports.
const Toolchain workspace.Toolchain = "go"

// ChangelogFile is the changelog of a module, in the directory of the module. Its headings record
// the versions of the module.
const ChangelogFile = changeset.Changelog

// versionHeading matches a heading of a changelog that names a version, such as ## 1.2.0, with the
// version in its first group.
var versionHeading = regexp.MustCompile(`(?m)^#{1,6}[ \t]+v?(\d+\.\d+\.\d+\S*)[ \t]*$`)

// Discoverer discovers the Go modules of a repository as the packages of [Toolchain].
//
// # Concurrency
//
// A Discoverer is safe for concurrent use when its Tags is.
type Discoverer struct {
	// Tags returns the tags of the repository at a directory, each with the commit that it names.
	// It must not be nil.
	Tags func(ctx context.Context, dir string) (map[string]string, error)
}

// Discover returns a package of [Toolchain] for each module of [Modules] of the repository at
// root, in the order of Modules: the module path as its Name, the directory of the module, the
// require lines of its go.mod on the other modules as requirements of [workspace.KindRuntime] with
// the version of the line, and the version of the module. The version is the higher of the highest
// version of a heading of its CHANGELOG.md and the highest version of its tags, as the package
// documentation states. A tag with build metadata names no version of a module, as the go command
// refuses it.
//
// It returns no package and no error for a repository without Go. It returns the error of opening
// root, the error of [Modules], the error of reading a changelog, with its file, and the error of
// d.Tags.
func (d Discoverer) Discover(ctx context.Context, root string) ([]workspace.Package, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	mods, err := modules(r)
	if err != nil || len(mods) == 0 {
		return nil, err
	}
	tags, err := d.Tags(ctx, root)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]bool, len(mods))
	for _, m := range mods {
		paths[m.Path] = true
	}
	pkgs := make([]workspace.Package, 0, len(mods))
	for _, m := range mods {
		v, err := changelogVersion(r, m.Dir)
		if err != nil {
			return nil, err
		}
		if tagged := tagVersion(tags, m.Dir); tagged.Compare(v) > 0 {
			v = tagged
		}
		p := workspace.Package{Name: m.Path, Toolchain: Toolchain, Dir: m.Dir, Version: v}
		for _, r := range m.File.Require {
			if paths[r.Mod.Path] && r.Mod.Path != m.Path {
				p.Deps = append(p.Deps, workspace.Dependency{
					Name: r.Mod.Path, Kind: workspace.KindRuntime, Req: r.Mod.Version,
				})
			}
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// changelogVersion returns the highest version that a heading of the changelog of the module in
// the directory dir of the repository of r names, and the zero version for a module without a
// changelog or without such a heading. It returns the error of reading the changelog.
func changelogVersion(r *os.Root, dir string) (version.Version, error) {
	file := path.Join(dir, ChangelogFile)
	data, err := r.ReadFile(filepath.FromSlash(file))
	if errors.Is(err, fs.ErrNotExist) {
		return version.Version{}, nil
	}
	if err != nil {
		return version.Version{}, fmt.Errorf("workspace: read %s: %w", file, err)
	}
	var highest version.Version
	for _, m := range versionHeading.FindAllSubmatch(data, -1) {
		if v, err := version.Parse(string(m[1])); err == nil && v.Compare(highest) > 0 {
			highest = v
		}
	}
	return highest, nil
}

// tagVersion returns the highest version that a tag of tags names for the module in the directory
// dir, and the zero version for none: a tag v<version> for the module at the root, and
// <dir>/v<version> for any other.
func tagVersion(tags map[string]string, dir string) version.Version {
	prefix := "v"
	if dir != "." {
		prefix = dir + "/v"
	}
	var highest version.Version
	for name := range tags {
		rest, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		if v, err := version.Parse(rest); err == nil && v.Build == "" && v.Compare(highest) > 0 {
			highest = v
		}
	}
	return highest
}
