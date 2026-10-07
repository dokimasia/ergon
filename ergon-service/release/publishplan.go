// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// The kinds of an entry of a publish plan, as changesets v3 spells them.
const (
	// KindPublish is a package that a publish uploads to its registry and then tags.
	KindPublish = "publish"

	// KindTagOnly is a package that a publish tags alone, such as a Go module, whose tag is its
	// release.
	KindTagOnly = "tag-only"
)

// publishPlanVersion is the version of the format of a publish plan.
const publishPlanVersion = 1

// PublishEntry is a package that a publish uploads or tags.
type PublishEntry struct {
	// Kind is [KindPublish] or [KindTagOnly].
	Kind string `json:"kind"`

	// Toolchain is the toolchain of the package.
	Toolchain workspace.Toolchain `json:"toolchain"`

	// Name is the name of the package in a changeset.
	Name string `json:"name"`

	// Tag is the tag of the package at Version.
	Tag string `json:"tag"`

	// Version is the version that the publish releases.
	Version version.Version `json:"version"`
}

// PublishPlan is the plan of a publish, in the format of changesets v3 with the toolchain and the
// tag of each entry: chunks of entries in dependency order, each chunk after the chunks of the
// packages that its packages require outside the dev section.
type PublishPlan struct {
	// Plan are the chunks of the plan.
	Plan [][]PublishEntry `json:"plan"`

	// Version is the version of the format, 1.
	Version int `json:"version"`
}

// NewPublishPlan returns the publish plan of the packages of g under c, with tags the tags of the
// repository and the commit of each:
//
//   - An entry of [KindPublish] for each package whose toolchain has a publisher and whose
//     registry does not have its version, as the publisher reports.
//   - An entry of [KindTagOnly] for each package of a toolchain without a publisher, with a
//     version that is not zero and without its tag in tags.
//
// A private package has an entry of [KindTagOnly] under privatePackages.tag alone, when tags lacks
// its tag. The tag of a package is the tag of the tagger of its toolchain, or else the tag of
// changesets: v<version> in a repository with one package, and <name>@<version> otherwise.
//
// It returns the error of a publisher, with the name of its package.
func NewPublishPlan(ctx context.Context, g *Graph, c *Config, tags map[string]string) (PublishPlan, error) {
	entries := map[int]PublishEntry{}
	for i := range g.pkgs {
		p := &g.pkgs[i]
		e := PublishEntry{
			Kind: KindTagOnly, Toolchain: p.Toolchain, Name: g.names[i], Tag: g.tag(i, p.Version), Version: p.Version,
		}
		_, tagged := tags[e.Tag]
		switch publisher := g.roles[i].publisher; {
		case p.Private:
			if c.PrivateTag && !tagged {
				entries[i] = e
			}
		case publisher != nil:
			published, err := publisher.Published(ctx, p)
			if err != nil {
				return PublishPlan{}, fmt.Errorf("release: the registry of %s: %w", g.names[i], err)
			}
			if !published {
				e.Kind = KindPublish
				entries[i] = e
			}
		case !p.Version.IsZero() && !tagged:
			entries[i] = e
		}
	}
	return PublishPlan{Version: publishPlanVersion, Plan: g.chunks(entries)}, nil
}

// Empty reports whether p has no entry.
func (p *PublishPlan) Empty() bool {
	return !slices.ContainsFunc(p.Plan, func(chunk []PublishEntry) bool { return len(chunk) > 0 })
}

// check returns an error that wraps [ErrPublishPlan] for the first entry of p that does not fit g:
// an entry of a package that g does not have, and an entry of [KindPublish] of a package whose
// toolchain has no publisher.
func (p *PublishPlan) check(g *Graph) error {
	for _, chunk := range p.Plan {
		for k := range chunk {
			i, ok := g.index[chunk[k].Name]
			switch {
			case !ok:
				return fmt.Errorf("%w: the package %s, which the repository does not have", ErrPublishPlan,
					chunk[k].Name)
			case chunk[k].Kind == KindPublish && g.roles[i].publisher == nil:
				return fmt.Errorf("%w: the upload of %s, whose toolchain has no registry", ErrPublishPlan,
					chunk[k].Name)
			}
		}
	}
	return nil
}

// chunks returns entries in chunks of dependency order: each entry in the chunk after the last
// chunk of an entry whose package it requires outside the dev section, in the order of the
// packages within a chunk. The entries of packages that require each other share the chunk after
// the chunks of every other entry that they require.
func (g *Graph) chunks(entries map[int]PublishEntry) [][]PublishEntry {
	// A plan without entries encodes as [], as changesets writes it.
	out := [][]PublishEntry{}
	done := map[int]bool{}
	for len(done) < len(entries) {
		var chunk []int
		for i := range g.pkgs {
			if _, ok := entries[i]; ok && !done[i] && g.ready(i, entries, done) {
				chunk = append(chunk, i)
			}
		}
		if len(chunk) == 0 {
			for i := range g.pkgs {
				if _, ok := entries[i]; ok && !done[i] {
					chunk = append(chunk, i)
				}
			}
		}
		row := make([]PublishEntry, 0, len(chunk))
		for _, i := range chunk {
			done[i] = true
			row = append(row, entries[i])
		}
		out = append(out, row)
	}
	return out
}

// ready reports whether every package that the package at index i requires outside the dev section
// and that has an entry of entries is in done.
func (g *Graph) ready(i int, entries map[int]PublishEntry, done map[int]bool) bool {
	for j := range g.pkgs {
		if _, ok := entries[j]; ok && j != i && !done[j] && slices.Contains(g.shipped[j], i) {
			return false
		}
	}
	return true
}

// tag returns the tag of the package at index i at v: the tag of the tagger of its toolchain, or
// else v<version> in a repository with one package and <name>@<version> otherwise, as changesets
// names them.
func (g *Graph) tag(i int, v version.Version) string {
	if tagger := g.roles[i].tagger; tagger != nil {
		return tagger.Tag(&g.pkgs[i], v)
	}
	if len(g.pkgs) == 1 {
		return "v" + v.String()
	}
	return g.pkgs[i].Name + "@" + v.String()
}
