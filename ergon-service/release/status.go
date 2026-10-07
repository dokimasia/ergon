// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/vcs"
)

// Status is the state of the changesets of a branch against a ref of its base, as changeset status
// reports it.
type Status struct {
	// Plan is the release plan of the changesets that the branch added.
	Plan Plan `json:"plan"`

	// Changed are the names of the packages whose files the branch changed, as changedFilePatterns
	// counts them, in the order of the graph. A skipped package is not among them.
	Changed []string `json:"changed"`

	// Uncovered are the names of the changed packages that no changeset of the branch names, in the
	// order of Changed, and none when the branch added a changeset that names no package.
	Uncovered []string `json:"uncovered"`

	// Broken are the requirements of a package on another package of the repository that exclude the
	// current version of the other, each as a sentence.
	Broken []string `json:"broken"`
}

// NewStatus returns the status of the branch of the working tree at root against since, a ref of
// the base branch, with sets the changesets of the working tree:
//
//   - A changeset counts as added when the branch adds or changes its file since the merge base.
//   - A file that the branch changes belongs to the package with the longest directory that
//     contains it, and changes the package when the globs of changedFilePatterns admit its path
//     relative to that directory. A changeset file changes no package.
//
// It returns the error of git, which wraps [vcs.ErrGit], and the error of [NewPlan].
func NewStatus(ctx context.Context, root string, g *Graph, c *Config, sets []changeset.Changeset, since string) (
	Status, error,
) {
	changed, err := vcs.Changed(ctx, root, since)
	if err != nil {
		return Status{}, err
	}
	var added []changeset.Changeset
	for k := range sets {
		if slices.Contains(changed, changesetPath(sets[k].ID)) {
			added = append(added, sets[k])
		}
	}
	plan, err := NewPlan(g, c, added)
	if err != nil {
		return Status{}, err
	}
	s := Status{Plan: plan, Broken: g.broken()}
	changes := map[int]bool{}
	for _, file := range changed {
		if i, ok := g.owner(file); ok && !strings.HasPrefix(file, changeset.Dir+"/") {
			rel := strings.TrimPrefix(file, g.pkgs[i].Dir+"/")
			changes[i] = changes[i] || matches(c.ChangedFilePatterns, rel)
		}
	}
	empty := slices.ContainsFunc(added, func(s changeset.Changeset) bool { return len(s.Releases) == 0 })
	for i := range g.pkgs {
		if !changes[i] || c.skips(g.names[i], &g.pkgs[i]) {
			continue
		}
		s.Changed = append(s.Changed, g.names[i])
		named := slices.ContainsFunc(added, func(s changeset.Changeset) bool {
			return slices.ContainsFunc(s.Releases, func(r changeset.Release) bool { return r.Name == g.names[i] })
		})
		if !named && !empty {
			s.Uncovered = append(s.Uncovered, g.names[i])
		}
	}
	return s, nil
}

// owner returns the index of the package whose directory is the longest that contains file, and
// reports whether a package contains it.
func (g *Graph) owner(file string) (int, bool) {
	at, longest := -1, -1
	for i := range g.pkgs {
		dir := g.pkgs[i].Dir
		inside := dir == "." || file == dir || strings.HasPrefix(file, dir+"/")
		if inside && len(dir) > longest {
			at, longest = i, len(dir)
		}
	}
	return at, at >= 0
}

// broken returns a sentence for each requirement of a package of g on another of its toolchain that
// the versioner resolves as excluding the current version of the other, in the order of the
// packages and of their requirements.
func (g *Graph) broken() []string {
	var out []string
	for i := range g.pkgs {
		for _, d := range g.pkgs[i].Deps {
			j, ok := g.sibling(i, d.Name)
			if !ok {
				continue
			}
			resolution, err := g.roles[i].versioner.Resolve(d.Req, g.pkgs[j].Version)
			if err == nil && resolution == language.ResolutionExcluded {
				out = append(out, fmt.Sprintf("%s requires %s %s, which excludes its version %s", g.names[i], d.Name,
					d.Req, g.pkgs[j].Version))
			}
		}
	}
	return out
}
