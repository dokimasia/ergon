// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// ErrChangeset is the error for a changeset that names a package that the repository does not
// have, and for a changeset that names a skipped and a released package together.
var ErrChangeset = errors.New("release: invalid changeset")

// Release is the release of one package in a [Plan]. Its JSON form has the keys of changesets:
// name, type, changesets, oldVersion and newVersion.
type Release struct {
	// Name is the name of the package in a changeset.
	Name string `json:"name"`

	// Bump is the level of the release. A release at [version.BumpNone] changes the requirements
	// of the package and not its version.
	Bump version.Bump `json:"type"`

	// Changesets are the IDs of the changesets that name the package, in the order of the plan's
	// changesets, and empty for a release that a dependency or a group causes.
	Changesets []string `json:"changesets"`

	// Old is the version that the bump starts from: the version of the package, or the highest
	// version of its fixed or linked group.
	Old version.Version `json:"oldVersion"`

	// New is Old at Bump, and Old for [version.BumpNone].
	New version.Version `json:"newVersion"`
}

// Plan is the releases that a set of changesets causes, as changesets 3.0.3 plans them. Its JSON
// form is the release plan of changeset status --output, with the keys changesets and releases.
type Plan struct {
	// Changesets are the changesets of the plan, in the order of the files.
	Changesets []changeset.Changeset `json:"changesets"`

	// Releases are the releases of the plan: first the packages that the changesets name, in the
	// order of the changesets, then each package that a dependency or a group releases, in the
	// order in which the planner reached it.
	Releases []Release `json:"releases"`
}

// release is a release that the planner has not yet versioned.
type release struct {
	// changesets are the IDs of the changesets that name the package.
	changesets []string

	// bump is the level of the release.
	bump version.Bump

	// old is the version that the bump starts from.
	old version.Version

	// pkg is the index of the package in the graph.
	pkg int
}

// planner is the state of one call of [NewPlan]: the releases in the order of their first
// insertion, as a Map of JavaScript keeps them.
type planner struct {
	// g is the graph of the packages.
	g *Graph

	// c is the configuration of the release.
	c *Config

	// at maps the index of a package to the position of its release in order.
	at map[int]int

	// order are the releases. Replacing the release of a package keeps its position.
	order []*release
}

// NewPlan returns the plan of sets for the packages of g under c, by the rules of changesets
// 3.0.3, with the dependents of a release decided by how their consumers resolve a requirement:
//
//   - Each package that a changeset names takes the highest level among the changesets that name
//     it. A skipped package, as [Config] states, takes none.
//   - Until a pass changes nothing: a dependent of a released package that is not released yet
//     takes a patch when its requirement outside the dev section excludes or pins the new version,
//     so that a consumer of the dependent would not receive it, or under updateInternalDependents
//     always, and a dependent through the dev section alone takes none; each fixed group takes the
//     highest level and the highest version of its members; and each linked group gives its
//     released members the highest level of them and the highest version of the group.
//   - Each new version is the old version at the level, and the versioner of the package's
//     toolchain validates it.
//
// A versioner that resolves no requirement as pinned, as npm's, makes the plan of changesets 3.0.3
// for the same files. A versioner that pins every newer version, as Go's, releases every package
// that requires a released package, directly or through other packages.
//
// It returns an error that wraps [ErrChangeset] and names the file and the line for a changeset
// that names a package that g does not have, and for a changeset that names a skipped and a
// released package. It returns an error for a version past its bound, and the error of a versioner
// that refuses a new version, each with the name of the package.
func NewPlan(g *Graph, c *Config, sets []changeset.Changeset) (Plan, error) {
	if err := checkChangesets(g, c, sets); err != nil {
		return Plan{}, err
	}
	p := planner{g: g, c: c, at: map[int]int{}}
	p.flatten(sets)
	for {
		dependents, err := p.dependents()
		if err != nil {
			return Plan{}, err
		}
		fixed := p.fixed()
		linked := p.linked()
		if !dependents && !fixed && !linked {
			break
		}
	}
	plan := Plan{Changesets: sets, Releases: make([]Release, 0, len(p.order))}
	for _, r := range p.order {
		next, err := r.old.Bump(r.bump)
		if err != nil {
			return Plan{}, fmt.Errorf("release: %s: %w", g.names[r.pkg], err)
		}
		if r.bump != version.BumpNone {
			if err := g.roles[r.pkg].versioner.Validate(&g.pkgs[r.pkg], next); err != nil {
				return Plan{}, fmt.Errorf("release: %s at %s: %w", g.names[r.pkg], next, err)
			}
		}
		plan.Releases = append(plan.Releases, Release{
			Name: g.names[r.pkg], Bump: r.bump, Changesets: r.changesets, Old: r.old, New: next,
		})
	}
	return plan, nil
}

// checkChangesets returns an error that wraps [ErrChangeset] for the first changeset of sets that
// names a package that g does not have, or that names a skipped and a released package.
func checkChangesets(g *Graph, c *Config, sets []changeset.Changeset) error {
	for _, s := range sets {
		var skipped, released []string
		for _, r := range s.Releases {
			i, ok := g.index[r.Name]
			if !ok {
				return fmt.Errorf("%w: %s/%s%s:%d: the package %q, which the repository does not have", ErrChangeset,
					changeset.Dir, s.ID, changeset.Ext, r.Line, r.Name)
			}
			if c.skips(r.Name, &g.pkgs[i]) {
				skipped = append(skipped, r.Name)
			} else {
				released = append(released, r.Name)
			}
		}
		if len(skipped) > 0 && len(released) > 0 {
			return fmt.Errorf("%w: %s/%s%s names the skipped packages %s and the released packages %s: write one "+
				"changeset for each", ErrChangeset, changeset.Dir, s.ID, changeset.Ext, strings.Join(skipped, ", "),
				strings.Join(released, ", "))
		}
	}
	return nil
}

// flatten adds a release for each package that sets name and c does not skip, at the highest level
// among the changesets that name it.
func (p *planner) flatten(sets []changeset.Changeset) {
	for _, s := range sets {
		for _, r := range s.Releases {
			i := p.g.index[r.Name]
			if p.c.skips(r.Name, &p.g.pkgs[i]) {
				continue
			}
			if slot, ok := p.at[i]; ok {
				existing := p.order[slot]
				existing.bump = existing.bump.Max(r.Bump)
				existing.changesets = append(existing.changesets, s.ID)
				continue
			}
			p.set(&release{pkg: i, bump: r.Bump, old: p.g.pkgs[i].Version, changesets: []string{s.ID}})
		}
	}
}

// dependents releases the dependents of every release, and of every release that it adds, and
// reports whether it added or replaced one. It returns the error of a version past its bound.
func (p *planner) dependents() (bool, error) {
	updated := false
	queue := slices.Clone(p.order)
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, d := range p.g.dependents[next.pkg] {
			bump, ok, err := p.dependentBump(next, d)
			if err != nil {
				return false, err
			}
			if !ok {
				continue
			}
			if slot, released := p.at[d]; released && p.order[slot].bump == bump {
				continue
			}
			updated = true
			r := &release{pkg: d, bump: bump, old: p.g.pkgs[d].Version, changesets: []string{}}
			queue = append(queue, r)
			p.set(r)
		}
	}
	return updated, nil
}

// dependentBump returns the level that the release next gives the package at index d, which
// requires it, and reports whether next gives it one: none for a skipped package, a patch for a
// requirement outside the dev section that excludes or pins the new version, so that a consumer of
// the dependent would not receive it, or for every requirement under updateInternalDependents
// always, and none for such a requirement in the dev section alone. A release at none and a
// dependent that is released already at another level than none give nothing. It returns the
// error of a version past its bound.
func (p *planner) dependentBump(next *release, d int) (version.Bump, bool, error) {
	if p.c.skips(p.g.names[d], &p.g.pkgs[d]) {
		return version.BumpNone, true, nil
	}
	var bump version.Bump
	ok := false
	for _, req := range p.g.requirements(d, next.pkg) {
		if next.bump == version.BumpNone {
			continue
		}
		if slot, released := p.at[d]; released && p.order[slot].bump != version.BumpNone {
			continue
		}
		if p.c.UpdateInternalDependents != DependentsAlways {
			newVersion, err := next.old.Bump(next.bump)
			if err != nil {
				return "", false, fmt.Errorf("release: %s: %w", p.g.names[next.pkg], err)
			}
			// A requirement that the versioner does not read selects no version, as semver.satisfies
			// of node-semver reports false for a range that it does not read.
			resolution, err := p.g.roles[d].versioner.Resolve(req.Req, newVersion)
			if err == nil && resolution == language.ResolutionSelected {
				continue
			}
		}
		switch {
		case req.Kind != workspace.KindDev && bump != version.BumpMajor && bump != version.BumpMinor:
			bump, ok = version.BumpPatch, true
		case req.Kind == workspace.KindDev && !bump.AtLeast(version.BumpPatch):
			bump, ok = version.BumpNone, true
		}
	}
	return bump, ok, nil
}

// fixed gives the members of each fixed group with a released member the highest level and the
// highest version of the group, adds a release for each member that c does not skip and that has
// none, and reports whether it changed a release.
func (p *planner) fixed() bool {
	updated := false
	for _, group := range p.c.Fixed {
		releasing := p.releasing(group)
		if len(releasing) == 0 {
			continue
		}
		bump, highest := highestBump(releasing), p.highestVersion(group)
		for _, name := range group {
			i := p.g.index[name]
			if p.c.skips(name, &p.g.pkgs[i]) {
				continue
			}
			slot, ok := p.at[i]
			if !ok {
				updated = true
				p.set(&release{pkg: i, bump: bump, old: highest, changesets: []string{}})
				continue
			}
			r := p.order[slot]
			if r.bump != bump {
				updated, r.bump = true, bump
			}
			if r.old != highest {
				updated, r.old = true, highest
			}
		}
	}
	return updated
}

// linked gives the released members of each linked group the highest level among them and the
// highest version of the group, and reports whether it changed a release.
func (p *planner) linked() bool {
	updated := false
	for _, group := range p.c.Linked {
		releasing := p.releasing(group)
		if len(releasing) == 0 {
			continue
		}
		bump, highest := highestBump(releasing), p.highestVersion(group)
		for _, r := range releasing {
			if r.bump != bump {
				updated, r.bump = true, bump
			}
			if r.old != highest {
				updated, r.old = true, highest
			}
		}
	}
	return updated
}

// releasing returns the releases of the members of group at a level above none, in the order of
// the releases.
func (p *planner) releasing(group []string) []*release {
	var out []*release
	for _, r := range p.order {
		if r.bump != version.BumpNone && slices.Contains(group, p.g.names[r.pkg]) {
			out = append(out, r)
		}
	}
	return out
}

// highestVersion returns the highest current version among the members of group, the first of
// them among versions of the same precedence.
func (p *planner) highestVersion(group []string) version.Version {
	var highest version.Version
	for k, name := range group {
		v := p.g.pkgs[p.g.index[name]].Version
		if k == 0 || v.Compare(highest) > 0 {
			highest = v
		}
	}
	return highest
}

// set puts r in place of the release of its package, at the same position, or adds it at the end.
func (p *planner) set(r *release) {
	if slot, ok := p.at[r.pkg]; ok {
		p.order[slot] = r
		return
	}
	p.at[r.pkg] = len(p.order)
	p.order = append(p.order, r)
}

// highestBump returns the highest level of releases.
func highestBump(releases []*release) version.Bump {
	bump := version.BumpNone
	for _, r := range releases {
		bump = bump.Max(r.bump)
	}
	return bump
}
