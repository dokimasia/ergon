// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"cmp"
	"fmt"
	"path"
	"slices"

	"go.dokimi.dev/ergon/core/workspace"
)

// rootDir is the directory of the root of the repository, relative to the root.
const rootDir = "."

// Requirement is a requirement of a crate on the crate of a path: a dependency of its Cargo.toml
// with the key path, of its own or through workspace.dependencies of the root of its workspace.
type Requirement struct {
	// Name is the name of the required crate: the key package of the dependency, or its key.
	Name string

	// Dir is the directory of the required crate, relative to the root of the repository and
	// slash-separated, or the absolute directory of an absolute path.
	Dir string

	// Req is the version requirement, and empty without one.
	Req string

	// File is the manifest that states Req: the Cargo.toml of the crate, or the Cargo.toml at the
	// root for an inherited dependency. It is relative to the root of the repository and
	// slash-separated.
	File string

	// Kind is the kind of the requirement in the manifest of the crate.
	Kind workspace.Kind

	// Span is the literal of Req in File, with its quotes, and the zero Span without a version.
	Span Span
}

// Requirements returns the requirements of the crate of the manifest m in the directory dir on the
// crates of a path, in the order of the dependencies of m. dir is relative to the root of the
// repository and slash-separated, and top is the manifest at the root. An inherited dependency
// takes its path, its version and its key package from workspace.dependencies of top, and keeps the
// kind of its dependency in m. A path is relative to the directory of the manifest that states it.
//
// It returns an error that wraps [ErrManifest] for an inherited dependency that
// workspace.dependencies of top does not have.
func Requirements(top, m *Manifest, dir string) ([]Requirement, error) {
	var out []Requirement
	for i := range m.Dependencies {
		d := &m.Dependencies[i]
		file, from, source := path.Join(dir, ManifestFile), dir, d
		if d.Workspace {
			k := slices.IndexFunc(top.WorkspaceDependencies, func(w Dependency) bool { return w.Key == d.Key })
			if k < 0 {
				return nil, fmt.Errorf("%w: %s: the dependency %s inherits from workspace.dependencies, which lacks it",
					ErrManifest, file, d.Key)
			}
			file, from, source = ManifestFile, rootDir, &top.WorkspaceDependencies[k]
		}
		if source.Path == "" {
			continue
		}
		target := path.Clean(source.Path)
		if !path.IsAbs(target) {
			target = path.Join(from, target)
		}
		out = append(out, Requirement{
			Name: cmp.Or(source.Package, d.Key),
			Dir:  target,
			Req:  source.Version,
			File: file,
			Kind: d.Kind,
			Span: source.VersionSpan,
		})
	}
	return out, nil
}
