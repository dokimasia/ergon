// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// ErrPackages is the error for packages that ergon cannot release: two packages of one toolchain
// with the same name, and a package of a toolchain without a [language.Versioner].
var ErrPackages = errors.New("release: invalid packages")

// sections are the kinds of a requirement in the order in which changesets reads the sections of a
// manifest. When a package requires another in several sections, the requirement of the last
// section decides whether the package is a dependent of the other.
var sections = []workspace.Kind{workspace.KindRuntime, workspace.KindDev, workspace.KindPeer, workspace.KindOptional}

// roles are the release roles and the facts of the toolchain of a package. A role that the
// toolchain does not implement is nil, except the versioner, which every toolchain with packages
// implements.
type roles struct {
	// versioner resolves and rewrites requirements, refuses versions and writes the edits.
	versioner language.Versioner

	// tagger names the tags of the toolchain, or is nil for the tags of changesets.
	tagger language.Tagger

	// packer builds the artifacts of a registry, or is nil for a toolchain without artifacts.
	packer language.Packer

	// publisher uploads the artifacts, or is nil for a toolchain that publishes by its tags.
	publisher language.Publisher

	// changelogVersion reports that a package records its version in its CHANGELOG.md, as
	// [language.Toolchain] states.
	changelogVersion bool
}

// Graph is the packages of a repository, the name that a changeset gives each of them, the release
// roles of their toolchains, and the dependents of each package. A package is a dependent of
// another of its toolchain when the versioner of the toolchain does not resolve its requirement as
// excluding the current version of the other.
//
// # Concurrency
//
// A Graph is not modified after [NewGraph] returns it, so it is safe for concurrent use.
type Graph struct {
	// index maps the name of each package in a changeset to its index in pkgs.
	index map[string]int

	// byName maps each toolchain and the Name of each of its packages to the package's index in
	// pkgs.
	byName map[workspace.Toolchain]map[string]int

	// pkgs are the packages, in the order of discovery.
	pkgs []workspace.Package

	// names are the names of pkgs in a changeset: the Name, or <toolchain>:<name> for a Name that
	// another package has too.
	names []string

	// roles are the roles of the toolchain of each of pkgs. The packages of one toolchain share
	// one value.
	roles []*roles

	// dependents are the indexes of the packages that require each of pkgs, in every section, in
	// ascending order.
	dependents [][]int

	// shipped are the indexes of the packages that require each of pkgs outside the dev section,
	// in ascending order: the requirements of the code that a package ships.
	shipped [][]int
}

// NewGraph returns the graph of pkgs, with the release roles and the facts that the toolchain of
// each package registers in c. The packages keep their order.
//
// It returns an error that wraps [ErrPackages] for a package whose toolchain registers no
// [language.Versioner], and for two packages that would have the same name in a changeset: two
// packages of one toolchain with the same Name.
func NewGraph(c *language.Catalog, pkgs []workspace.Package) (*Graph, error) {
	g := &Graph{index: make(map[string]int, len(pkgs)), byName: map[workspace.Toolchain]map[string]int{}, pkgs: pkgs}
	shared := map[string]int{}
	for i := range pkgs {
		p := &pkgs[i]
		shared[p.Name]++
		if g.byName[p.Toolchain] == nil {
			g.byName[p.Toolchain] = map[string]int{}
		}
		g.byName[p.Toolchain][p.Name] = i
	}
	byToolchain := map[workspace.Toolchain]*roles{}
	for i := range pkgs {
		p := &pkgs[i]
		r, ok := byToolchain[p.Toolchain]
		if !ok {
			v, found := language.ToolchainRole[language.Versioner](c, p.Toolchain)
			if !found {
				return nil, fmt.Errorf("%w: %q, whose toolchain %s registers no versioner", ErrPackages, p.Name,
					p.Toolchain)
			}
			// A toolchain with a role is registered, so the catalog has it.
			t, _ := c.Toolchain(p.Toolchain)
			r = &roles{versioner: v, changelogVersion: t.ChangelogVersion}
			r.tagger, _ = language.ToolchainRole[language.Tagger](c, p.Toolchain)
			r.packer, _ = language.ToolchainRole[language.Packer](c, p.Toolchain)
			r.publisher, _ = language.ToolchainRole[language.Publisher](c, p.Toolchain)
			byToolchain[p.Toolchain] = r
		}
		name := p.Name
		if shared[p.Name] > 1 {
			name = string(p.Toolchain) + ":" + p.Name
		}
		if _, taken := g.index[name]; taken {
			return nil, fmt.Errorf("%w: two packages named %q", ErrPackages, name)
		}
		g.index[name] = i
		g.names = append(g.names, name)
		g.roles = append(g.roles, r)
	}
	g.dependents = g.edges(true)
	g.shipped = g.edges(false)
	return g, nil
}

// Discover returns the graph of the packages that the toolchains of c discover in the repository at
// root, in the order of the catalog and of each discovery. It skips a toolchain without
// [language.Toolchain.Discover].
//
// It returns the error of a discovery, wrapped with the name of its toolchain, and the error of
// [NewGraph].
func Discover(ctx context.Context, c *language.Catalog, root string) (*Graph, error) {
	var pkgs []workspace.Package
	for t := range c.Toolchains() {
		if t.Discover == nil {
			continue
		}
		found, err := t.Discover(ctx, root)
		if err != nil {
			return nil, fmt.Errorf("release: discover the packages of the toolchain %s: %w", t.Name, err)
		}
		pkgs = append(pkgs, found...)
	}
	return NewGraph(c, pkgs)
}

// Packages returns the packages of g, in their order. The caller does not modify them.
func (g *Graph) Packages() []workspace.Package {
	return g.pkgs
}

// edges returns, for each package, the indexes of the packages that require it, in ascending
// order: in every section with dev set, and outside the dev section otherwise. A package requires
// another of its toolchain when the requirement of the last section that names the other pins or
// selects the current version of the other. A requirement that the versioner does not read, such
// as link:../a of npm, makes no edge, as changesets counts no dependent through a range that it
// does not read.
func (g *Graph) edges(dev bool) [][]int {
	out := make([][]int, len(g.pkgs))
	for i := range g.pkgs {
		p := &g.pkgs[i]
		decided := map[int]workspace.Dependency{}
		var order []int
		for _, kind := range sections {
			if kind == workspace.KindDev && !dev {
				continue
			}
			for _, d := range p.Deps {
				j, ok := g.sibling(i, d.Name)
				if !ok || d.Kind != kind {
					continue
				}
				if _, seen := decided[j]; !seen {
					order = append(order, j)
				}
				decided[j] = d
			}
		}
		for _, j := range order {
			r, err := g.roles[i].versioner.Resolve(decided[j].Req, g.pkgs[j].Version)
			if err == nil && (r == language.ResolutionPinned || r == language.ResolutionSelected) {
				out[j] = append(out[j], i)
			}
		}
	}
	return out
}

// sibling returns the index of the package of the toolchain of the package at index i whose Name is
// name, and reports whether such a package other than i exists.
func (g *Graph) sibling(i int, name string) (int, bool) {
	j, ok := g.byName[g.pkgs[i].Toolchain][name]
	return j, ok && j != i
}

// requirements returns the requirements of the package at index i on the package at index j, in
// the order of sections: one per section that names it.
func (g *Graph) requirements(i, j int) []workspace.Dependency {
	var out []workspace.Dependency
	for _, kind := range sections {
		for _, d := range g.pkgs[i].Deps {
			if d.Kind == kind && d.Name == g.pkgs[j].Name {
				out = append(out, d)
			}
		}
	}
	return out
}
