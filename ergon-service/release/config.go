// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// ConfigPath is the path of the configuration of a release, relative to the root of the
// repository.
const ConfigPath = ".changeset/config.json"

// The changelog formats of changesets that ergon writes.
const (
	// ChangelogGit is the format of @changesets/cli/changelog: each entry is the summary of a
	// changeset with the short commit that added it.
	ChangelogGit = "@changesets/cli/changelog"

	// ChangelogGitModule is the module that @changesets/cli/changelog exports, which a
	// configuration may name instead.
	ChangelogGitModule = "@changesets/changelog-git"

	// ChangelogGitHub is the format of @changesets/changelog-github: each entry links the pull
	// request, the commit and the author of a changeset on GitHub.
	ChangelogGitHub = "@changesets/changelog-github"
)

// The values of the option updateInternalDependents.
const (
	// DependentsOutOfRange releases a dependent whose consumers would not receive the new version of
	// a package that it requires: its requirement excludes or pins the new version. For npm, whose
	// requirements pin nothing, it is the rule of changesets: the requirement excludes the version.
	DependentsOutOfRange = "out-of-range"

	// DependentsAlways releases every dependent of a released package.
	DependentsAlways = "always"
)

// ErrConfig is the error for a .changeset/config.json that ergon does not read, and for one whose
// groups and packages contradict each other.
var ErrConfig = errors.New("release: invalid .changeset/config.json")

// Changelog is the changelog format of a configuration. The zero value writes no changelog.
type Changelog struct {
	// Format is [ChangelogGit] or [ChangelogGitHub], and empty for no changelog.
	Format string

	// Repo is the repository on GitHub, as owner/name, of [ChangelogGitHub], or empty for the
	// repository of the workflow.
	Repo string

	// Template is the line of an entry of [ChangelogGitHub] with the tokens {summary}, {ref},
	// {pull}, {commit} and {authors}, or empty for the line of changesets.
	Template string

	// DisableThanks leaves out the author of each entry of [ChangelogGitHub].
	DisableThanks bool
}

// Config is the configuration of a release: .changeset/config.json, in the format of changesets
// 4.0.1 with its defaults. ergon reads every key of changesets that changes the release of a
// package of another language than JavaScript, and ignores the keys that only the tools of npm
// read, such as format and snapshot.
type Config struct {
	// Changelog is the format of the changelogs.
	Changelog Changelog

	// BaseBranch is the branch that status and add compare against, main by default.
	BaseBranch string

	// Access is the access of npm for a package without its own, restricted by default.
	Access string

	// UpdateInternalDependencies is the lowest level of a release whose new version a dependent's
	// requirement takes while the requirement still selects it: patch by default, or minor.
	UpdateInternalDependencies version.Bump

	// UpdateInternalDependents is [DependentsOutOfRange] or [DependentsAlways].
	UpdateInternalDependents string

	// ChangedFilePatterns are the doublestar globs of the files of a package, relative to its
	// directory, whose change counts for status: ** by default. A glob after ! removes the files
	// that it matches from those that an earlier glob matched.
	ChangedFilePatterns []string

	// Ignore are the names of the packages that are never released.
	Ignore []string

	// Fixed are the groups of packages that share one version and are released together: the
	// groups of the configuration, and the packages that share a [workspace.Package] Source.
	Fixed [][]string

	// Linked are the groups of packages whose released members share the highest level and the
	// highest version.
	Linked [][]string

	// Warnings are the names and globs of fixed and linked that match no package, which changesets
	// reports and accepts.
	Warnings []string

	// PrivateVersion releases the packages marked private, and PrivateTag tags them.
	PrivateVersion, PrivateTag bool

	// OnlyUpdatePeerDependentsWhenOutOfRange keeps a peer requirement that still selects the new
	// version of the package that it names.
	OnlyUpdatePeerDependentsWhenOutOfRange bool
}

// jsonNull is the JSON null, which changesets refuses where a key admits several types.
const jsonNull = "null"

// writtenConfig is .changeset/config.json as JSON states it. The keys that ergon does not read
// decode into nothing.
type writtenConfig struct {
	Unsafe struct {
		UpdateInternalDependents               *string `json:"updateInternalDependents"`
		OnlyUpdatePeerDependentsWhenOutOfRange bool    `json:"onlyUpdatePeerDependentsWhenOutOfRange"`
	} `json:"___experimentalUnsafeOptions_WILL_CHANGE_IN_PATCH"`
	Changelog                  json.RawMessage `json:"changelog"`
	Commit                     json.RawMessage `json:"commit"`
	PrivatePackages            json.RawMessage `json:"privatePackages"`
	BaseBranch                 *string         `json:"baseBranch"`
	Access                     *string         `json:"access"`
	UpdateInternalDependencies *string         `json:"updateInternalDependencies"`
	ChangedFilePatterns        []string        `json:"changedFilePatterns"`
	Ignore                     []string        `json:"ignore"`
	Fixed                      [][]string      `json:"fixed"`
	Linked                     [][]string      `json:"linked"`
}

// ParseConfig returns the configuration in data, the content of .changeset/config.json, for the
// packages of g. Each name and glob of ignore, fixed and linked becomes the names of the packages
// that it matches, as changesets matches them: a name passes the first glob that matches it, and a
// later glob after ! that matches it removes it again. An absent key takes the default of
// changesets.
//
// It returns an error that wraps [ErrConfig] for data that is no JSON object, for a key of a type
// that changesets does not accept, for a changelog other than false and the two formats of
// [Changelog], and for a commit other than false, because ergon makes no commit of its own. It
// joins the errors of the rules of changesets and of ergon into one: a package that two fixed or
// two linked groups name, a package that a fixed and a linked group name, a package that shares a
// Source with another and that a group of the configuration names, a package that requires a
// skipped package outside the dev section and is neither skipped nor private, privatePackages.tag
// without privatePackages.version, and changelog false in a repository with a package whose
// toolchain records versions in changelogs.
func ParseConfig(data []byte, g *Graph) (Config, error) {
	var w writtenConfig
	if err := json.Unmarshal(data, &w); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrConfig, err)
	}
	c := Config{
		BaseBranch:                 orDefault(w.BaseBranch, "main"),
		Access:                     orDefault(w.Access, "restricted"),
		UpdateInternalDependencies: version.Bump(orDefault(w.UpdateInternalDependencies, string(version.BumpPatch))),
		UpdateInternalDependents:   orDefault(w.Unsafe.UpdateInternalDependents, DependentsOutOfRange),
		ChangedFilePatterns:        w.ChangedFilePatterns,

		OnlyUpdatePeerDependentsWhenOutOfRange: w.Unsafe.OnlyUpdatePeerDependentsWhenOutOfRange,
	}
	if c.ChangedFilePatterns == nil {
		c.ChangedFilePatterns = []string{"**"}
	}
	if c.Access != "public" && c.Access != "restricted" {
		return Config{}, fmt.Errorf("%w: access %q, which is neither public nor restricted", ErrConfig, c.Access)
	}
	if c.UpdateInternalDependencies != version.BumpPatch && c.UpdateInternalDependencies != version.BumpMinor {
		return Config{}, fmt.Errorf("%w: updateInternalDependencies %q, which is neither patch nor minor", ErrConfig,
			c.UpdateInternalDependencies)
	}
	if c.UpdateInternalDependents != DependentsOutOfRange && c.UpdateInternalDependents != DependentsAlways {
		return Config{}, fmt.Errorf("%w: updateInternalDependents %q, which is neither out-of-range nor always",
			ErrConfig, c.UpdateInternalDependents)
	}
	var err error
	if c.Changelog, err = changelogOf(w.Changelog); err != nil {
		return Config{}, err
	}
	if len(w.Commit) > 0 && string(w.Commit) != "false" {
		return Config{}, fmt.Errorf("%w: commit %s, which is not false: ergon release version commits nothing",
			ErrConfig, w.Commit)
	}
	if c.PrivateVersion, c.PrivateTag, err = privateOf(w.PrivatePackages); err != nil {
		return Config{}, err
	}
	c.Ignore = globMatch(g.names, w.Ignore)
	for _, group := range w.Fixed {
		c.Fixed = append(c.Fixed, globMatch(g.names, group))
		c.Warnings = append(c.Warnings, unmatched("fixed", g.names, group)...)
	}
	for _, group := range w.Linked {
		c.Linked = append(c.Linked, globMatch(g.names, group))
		c.Warnings = append(c.Warnings, unmatched("linked", g.names, group)...)
	}
	sources := sourceGroups(g)
	if err := errors.Join(
		duplicates("fixed", c.Fixed),
		duplicates("linked", c.Linked),
		fixedAndLinked(c.Fixed, c.Linked),
		sourcesAndGroups(sources, slices.Concat(c.Fixed, c.Linked)),
		c.skippedDependents(g),
		c.privateTag(),
		c.changelogVersions(g),
	); err != nil {
		return Config{}, err
	}
	c.Fixed = append(c.Fixed, sources...)
	return c, nil
}

// skips reports whether a release leaves out p, whose name in a changeset is name: a package that
// ignore names, and a private package while privatePackages.version is false.
func (c *Config) skips(name string, p *workspace.Package) bool {
	return slices.Contains(c.Ignore, name) || (p.Private && !c.PrivateVersion)
}

// skippedDependents returns an error that wraps [ErrConfig] for each package of g that requires a
// skipped package outside the dev section and is neither skipped nor private, and nil for none or
// for a configuration that ignores no package and releases the private ones.
func (c *Config) skippedDependents(g *Graph) error {
	if len(c.Ignore) == 0 && c.PrivateVersion {
		return nil
	}
	var errs []error
	for j := range g.pkgs {
		if !c.skips(g.names[j], &g.pkgs[j]) {
			continue
		}
		for _, i := range g.shipped[j] {
			if c.skips(g.names[i], &g.pkgs[i]) || g.pkgs[i].Private {
				continue
			}
			errs = append(
				errs,
				fmt.Errorf("%w: %q requires the skipped package %q and is not skipped: add it to ignore",
					ErrConfig, g.names[i], g.names[j]),
			)
		}
	}
	return errors.Join(errs...)
}

// privateTag returns an error that wraps [ErrConfig] for privatePackages.tag without
// privatePackages.version, and nil otherwise.
func (c *Config) privateTag() error {
	if c.PrivateTag && !c.PrivateVersion {
		return fmt.Errorf("%w: privatePackages.tag, which privatePackages.version requires", ErrConfig)
	}
	return nil
}

// changelogVersions returns an error that wraps [ErrConfig] for changelog false in a repository
// with a package whose toolchain records versions in changelogs, as go.mod has no version field,
// and nil otherwise. Between the merge of a version pull request and its tag, the heading of the
// changelog is the only record of such a package's version.
func (c *Config) changelogVersions(g *Graph) error {
	if c.Changelog.Format != "" {
		return nil
	}
	for i, r := range g.roles {
		if r.changelogVersion {
			return fmt.Errorf("%w: changelog false, which %q cannot release under: its toolchain %s records its "+
				"version in its CHANGELOG.md", ErrConfig, g.names[i], g.pkgs[i].Toolchain)
		}
	}
	return nil
}

// changelogOf returns the changelog format that raw, the key changelog, states: the default of
// changesets for an absent key. It returns an error that wraps [ErrConfig] for a value that is not
// false, a module, or a module with its options, and for a module other than the formats of
// [Changelog].
func changelogOf(raw json.RawMessage) (Changelog, error) {
	if len(raw) == 0 {
		return Changelog{Format: ChangelogGit}, nil
	}
	if string(raw) == "false" {
		return Changelog{}, nil
	}
	var module string
	var options json.RawMessage
	if err := json.Unmarshal(raw, &module); err != nil || string(raw) == jsonNull {
		var tuple []json.RawMessage
		if json.Unmarshal(raw, &tuple) != nil || len(tuple) != 2 || json.Unmarshal(tuple[0], &module) != nil {
			return Changelog{}, fmt.Errorf("%w: changelog %s, which is neither false, a module nor a module and its "+
				"options", ErrConfig, raw)
		}
		options = tuple[1]
	}
	switch module {
	case ChangelogGit, ChangelogGitModule:
		return Changelog{Format: ChangelogGit}, nil
	case ChangelogGitHub:
		var o struct {
			Repo          string `json:"repo"`
			Template      string `json:"template"`
			DisableThanks bool   `json:"disableThanks"`
		}
		if options != nil {
			if err := json.Unmarshal(options, &o); err != nil {
				return Changelog{}, fmt.Errorf("%w: the options of changelog: %w", ErrConfig, err)
			}
		}
		return Changelog{
			Format:        ChangelogGitHub,
			Repo:          o.Repo,
			Template:      o.Template,
			DisableThanks: o.DisableThanks,
		}, nil
	}
	return Changelog{}, fmt.Errorf("%w: changelog %q, which is none of %s and %s: ergon runs no JavaScript",
		ErrConfig, module, ChangelogGit, ChangelogGitHub)
}

// privateOf returns privatePackages.version and privatePackages.tag of raw, the key
// privatePackages: a bool for both, or an object with either, and false for an absent key. It
// returns an error that wraps [ErrConfig] for any other value.
func privateOf(raw json.RawMessage) (bool, bool, error) {
	if len(raw) == 0 {
		return false, false, nil
	}
	var both bool
	if json.Unmarshal(raw, &both) == nil && string(raw) != jsonNull {
		return both, both, nil
	}
	var o struct {
		Version bool `json:"version"`
		Tag     bool `json:"tag"`
	}
	if err := json.Unmarshal(raw, &o); err != nil || string(raw) == jsonNull {
		return false, false, fmt.Errorf("%w: privatePackages %s, which is neither a bool nor an object", ErrConfig,
			raw)
	}
	return o.Version, o.Tag, nil
}

// sourceGroups returns the packages of g that share a non-empty Source, one group per Source with
// at least two packages, in the order of the first package of each group.
func sourceGroups(g *Graph) [][]string {
	var sources []string
	groups := map[string][]string{}
	for i := range g.pkgs {
		source := g.pkgs[i].Source
		if source == "" {
			continue
		}
		if _, ok := groups[source]; !ok {
			sources = append(sources, source)
		}
		groups[source] = append(groups[source], g.names[i])
	}
	var out [][]string
	for _, s := range sources {
		if len(groups[s]) > 1 {
			out = append(out, groups[s])
		}
	}
	return out
}

// sourcesAndGroups returns an error that wraps [ErrConfig] for each package of sources that a
// group of groups names, and nil for none: two fixed groups that share a package have no common
// version.
func sourcesAndGroups(sources, groups [][]string) error {
	var errs []error
	for _, name := range slices.Concat(sources...) {
		if slices.ContainsFunc(groups, func(group []string) bool { return slices.Contains(group, name) }) {
			errs = append(errs, fmt.Errorf("%w: the package %q, which shares the version of its manifest with other "+
				"packages and which a group names: remove it from the group", ErrConfig, name))
		}
	}
	return errors.Join(errs...)
}

// duplicates returns an error that wraps [ErrConfig] for each package that two groups of kind name,
// and nil for none.
func duplicates(kind string, groups [][]string) error {
	var seen []string
	var errs []error
	for _, group := range groups {
		for _, name := range group {
			if slices.Contains(seen, name) {
				errs = append(errs, fmt.Errorf("%w: the package %q, which two %s groups name", ErrConfig, name, kind))
			}
			seen = append(seen, name)
		}
	}
	return errors.Join(errs...)
}

// fixedAndLinked returns an error that wraps [ErrConfig] for each package that a fixed and a
// linked group name, and nil for none.
func fixedAndLinked(fixed, linked [][]string) error {
	all := slices.Concat(linked...)
	var errs []error
	for _, name := range slices.Concat(fixed...) {
		if slices.Contains(all, name) {
			errs = append(errs, fmt.Errorf("%w: the package %q, which a fixed and a linked group name", ErrConfig,
				name))
		}
	}
	return errors.Join(errs...)
}

// unmatched returns a warning for each name and glob of group that matches none of names. A glob
// after ! matches the names that the glob without it does not match.
func unmatched(kind string, names, group []string) []string {
	var out []string
	for _, pattern := range group {
		glob, negated := strings.CutPrefix(pattern, "!")
		found := slices.ContainsFunc(names, func(name string) bool {
			matched, _ := doublestar.Match(glob, name)
			return matched != negated
		})
		if !found {
			out = append(out, fmt.Sprintf("%s: the package or glob %q matches no package", kind, pattern))
		}
	}
	return out
}

// globMatch returns the names of names that patterns admit, as [matches] states, in the order of
// names.
func globMatch(names, patterns []string) []string {
	var out []string
	for _, name := range names {
		if matches(patterns, name) {
			out = append(out, name)
		}
	}
	return out
}

// matches reports whether patterns admit path, as changesets admits it: path passes the first
// doublestar glob that matches it, and a later glob after ! that matches it removes it, until a
// later glob without ! admits it again.
func matches(patterns []string, path string) bool {
	passed := false
	for _, pattern := range patterns {
		glob, negated := strings.CutPrefix(pattern, "!")
		matched, _ := doublestar.Match(glob, path)
		switch {
		case !passed && !negated && matched:
			passed = true
		case passed && negated && matched:
			passed = false
		}
	}
	return passed
}

// orDefault returns *s, or fallback for a nil s.
func orDefault(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
