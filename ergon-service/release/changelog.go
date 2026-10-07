// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// ChangelogFile is the name of the changelog of a package, in the directory of the package.
const ChangelogFile = "CHANGELOG.md"

// defaultServer is the address of GitHub when GITHUB_SERVER_URL is empty.
const defaultServer = "https://github.com"

// The patterns of the changelog of GitHub, as @changesets/changelog-github 1.0.1 writes them.
var (
	// repoPattern matches a repository on GitHub as owner/name, as @changesets/get-github-info
	// accepts it.
	repoPattern = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)

	// pullDirective matches the line of a summary that names its pull request, such as pr: #12.
	pullDirective = regexp.MustCompile(`(?im)^\s*(?:pr|pull|pull\s+request):\s*#?(\d+)`)

	// commitDirective matches the line of a summary that names its commit, such as commit: abc1234.
	commitDirective = regexp.MustCompile(`(?im)^\s*commit:\s*([^\s]+)`)

	// authorDirective matches a line of a summary that names an author, such as author: @login.
	authorDirective = regexp.MustCompile(`(?im)^\s*(?:author|user):\s*@?([^\s]+)`)

	// issueRef matches a Markdown link, which the changelog keeps, and a reference to an issue,
	// such as #123, which it links.
	issueRef = regexp.MustCompile(`\[.*?\]\(.*?\)|\B#([1-9]\d*)\b`)

	// templateToken matches a token of the template of a line, such as {summary}.
	templateToken = regexp.MustCompile(`\{(\w+)\}`)
)

// versionHeading matches a heading of a version in a changelog, such as ## 1.2.0, which an entry
// goes before.
var versionHeading = regexp.MustCompile(`(?m)^#{1,6}\s+\d+\.\d+`)

// sectionHeading matches a heading of a changelog, or the fence of a code block, whose headings a
// section skips.
var sectionHeading = regexp.MustCompile("(?m)^(#{1,6})\\s(.*)$|^(`{3,})")

// levelWord matches the first level that a heading of a changelog names.
var levelWord = regexp.MustCompile(`major|minor|patch`)

// levels are the levels of the sections of an entry, from the highest to the lowest.
var levels = []version.Bump{version.BumpMajor, version.BumpMinor, version.BumpPatch}

// ErrChangelog is the error for an entry that the changelog of GitHub cannot render: a repository
// that is not owner/name, no host to read the links from, and a template with a token that it does
// not define.
var ErrChangelog = errors.New("release: invalid changelog")

// Entry is the changelog entry that a release writes for one package.
type Entry struct {
	// Name is the name of the package in a changeset.
	Name string

	// Text is the entry from its heading, ## and the new version, to its last line, without a
	// newline at the end.
	Text string
}

// update is a requirement on another package that a release rewrites, as an entry lists it.
type update struct {
	// name is the Name of the required package.
	name string

	// version is the version that the requirement moves to.
	version version.Version
}

// renderer renders the lines of the entries of one plan in one changelog format.
type renderer struct {
	// host is the repository that the changelog of GitHub links to.
	host Host

	// commits maps the ID of each changeset to the commit that added its file, or to the empty
	// string for a file that no commit added.
	commits map[string]string

	// repo is the repository that the changelog of GitHub links to, as owner/name.
	repo string

	// server is the address of GitHub, without a slash at the end.
	server string

	// format is the changelog format of the configuration.
	format Changelog
}

// Entries returns the changelog entry of each release of plan above none, in the order of the
// plan, in the format of c.Changelog, and nil for a configuration without a changelog. commits maps
// the ID of each changeset to the commit that added its file. An entry renders as changesets 3.0.3
// renders it with @changesets/cli/changelog or @changesets/changelog-github 1.0.1:
//
//   - The heading ## and the new version.
//   - A section per level, ### Major Changes to ### Patch Changes, with a line for each changeset
//     that names the package at that level, in the order of the plan.
//   - Under Patch Changes, a line that lists each requirement on another package that the release
//     rewrites, after the links of the changesets of those packages.
//   - "No changes in this release." for an entry without a section.
//
// The changelog of GitHub reads the links of each commit and pull request from h.Forge. It returns
// an error that wraps [ErrChangelog] for the changelog of GitHub without h.Forge, for a repository
// that is not owner/name, from the options of the format or from h.Repo, and for a template with a
// token that it does not define. It returns the error of h.Forge.
func Entries(
	ctx context.Context, g *Graph, c *Config, plan *Plan, commits map[string]string, h Host,
) ([]Entry, error) {
	if c.Changelog.Format == "" {
		return nil, nil
	}
	if c.Changelog.Format == ChangelogGitHub && h.Forge == nil {
		return nil, fmt.Errorf("%w: the changelog of GitHub, which needs the host of the repository", ErrChangelog)
	}
	w := &renderer{
		host: h, commits: commits, format: c.Changelog, repo: cmp.Or(c.Changelog.Repo, h.Repo),
		server: strings.TrimRight(cmp.Or(h.Server, defaultServer), "/"),
	}
	var out []Entry
	for k := range plan.Releases {
		r := &plan.Releases[k]
		if r.Bump == version.BumpNone {
			continue
		}
		text, err := w.entry(ctx, g, c, plan, r)
		if err != nil {
			return nil, fmt.Errorf("release: the changelog of %s: %w", r.Name, err)
		}
		out = append(out, Entry{Name: r.Name, Text: text})
	}
	return out, nil
}

// Section returns the section of the version v in data, the content of a changelog, without its
// heading and without the blank lines around it, as changesets/action v2 reads it for the body of
// a release and of the version pull request. The section ends at the next heading of the same
// depth, and a heading inside a fenced code block counts for nothing. It also returns the highest
// level that a heading from the start of data to the end of the section names, and reports whether
// data has a heading of v.
func Section(data []byte, v version.Version) (string, version.Bump, bool) {
	text, want := string(data), v.String()
	highest := version.BumpNone
	start, depth, end, resume := -1, 0, len(text), 0
	for _, m := range sectionHeading.FindAllStringSubmatchIndex(text, -1) {
		if m[0] < resume {
			continue
		}
		if m[6] >= 0 {
			closing := strings.Index(text[m[1]:], "\n"+text[m[6]:m[7]])
			if closing < 0 {
				break
			}
			resume = m[1] + closing + 1 + m[7] - m[6]
			continue
		}
		heading := strings.TrimSpace(text[m[4]:m[5]])
		if word := levelWord.FindString(strings.ToLower(heading)); word != "" {
			highest = highest.Max(version.Bump(word))
		}
		if heading == want {
			start, depth = m[1], m[3]-m[2]
			continue
		}
		if start >= 0 && m[3]-m[2] == depth {
			end = m[0]
			break
		}
	}
	if start < 0 {
		return "", highest, false
	}
	return strings.TrimSpace(text[start:end]), highest, true
}

// entry returns the entry of the release r of plan.
func (w *renderer) entry(ctx context.Context, g *Graph, c *Config, plan *Plan, r *Release) (string, error) {
	sections := make([][]string, len(levels))
	for k := range plan.Changesets {
		s := &plan.Changesets[k]
		at := slices.IndexFunc(s.Releases, func(n changeset.Release) bool { return n.Name == r.Name })
		if at < 0 || s.Releases[at].Bump == version.BumpNone {
			continue
		}
		line, err := w.releaseLine(ctx, s)
		if err != nil {
			return "", err
		}
		level := slices.Index(levels, s.Releases[at].Bump)
		sections[level] = append(sections[level], line)
	}
	updated, sets := dependencyUpdates(g, c, plan, r)
	line, err := w.dependencyLine(ctx, sets, updated)
	if err != nil {
		return "", err
	}
	patch := slices.Index(levels, version.BumpPatch)
	sections[patch] = append(sections[patch], line)
	parts := []string{"## " + r.New.String()}
	for k, level := range levels {
		if section := renderSection(level, sections[k]); section != "" {
			parts = append(parts, section)
		}
	}
	if len(parts) == 1 {
		parts = append(parts, "No changes in this release.")
	}
	return strings.Join(parts, "\n\n"), nil
}

// releaseLine returns the line of the changeset s in the format of w.
func (w *renderer) releaseLine(ctx context.Context, s *changeset.Changeset) (string, error) {
	if w.format.Format == ChangelogGitHub {
		return w.githubReleaseLine(ctx, s)
	}
	summary := strings.Split(s.Summary, "\n")
	for k := range summary {
		summary[k] = strings.TrimRightFunc(summary[k], unicode.IsSpace)
	}
	line := "- "
	if commit := w.commits[s.ID]; commit != "" {
		line += short(commit) + ": "
	}
	line += summary[0]
	if len(summary) > 1 {
		line += "\n  " + strings.Join(summary[1:], "\n  ")
	}
	return line, nil
}

// githubReleaseLine returns the line of the changeset s in the format of
// @changesets/changelog-github: the links of its pull request, its commit and its authors, and its
// summary with each reference to an issue linked. A line of the summary such as pr: #12, commit:
// abc1234 or author: @login replaces the link that the commit of the changeset gives.
func (w *renderer) githubReleaseLine(ctx context.Context, s *changeset.Changeset) (string, error) {
	if err := w.checkRepo(); err != nil {
		return "", err
	}
	summary, pull := cut(pullDirective, s.Summary)
	summary, commit := cut(commitDirective, summary)
	var users []string
	summary = authorDirective.ReplaceAllStringFunc(summary, func(match string) string {
		users = append(users, authorDirective.FindStringSubmatch(match)[1])
		return ""
	})
	var commitLink, pullLink, authorLink string
	switch {
	case pull != "":
		number, err := strconv.Atoi(pull)
		if err != nil {
			return "", fmt.Errorf("%w: the pull request %s of %s: %w", ErrChangelog, pull, s.ID, err)
		}
		if pullLink, commitLink, authorLink, err = w.host.Forge.PullLinks(ctx, w.repo, number); err != nil {
			return "", err
		}
		if commit != "" {
			commitLink = "[`" + short(commit) + "`](" + w.server + "/" + w.repo + "/commit/" + commit + ")"
		}
	case commit != "" || w.commits[s.ID] != "":
		var err error
		sha := cmp.Or(commit, w.commits[s.ID])
		if commitLink, pullLink, authorLink, err = w.host.Forge.CommitLinks(ctx, w.repo, sha); err != nil {
			return "", err
		}
	}
	authors := authorLink
	switch {
	case w.format.DisableThanks:
		authors = ""
	case len(users) > 0:
		for k, u := range users {
			users[k] = fmt.Sprintf("[@%s](%s/%s)", u, w.server, u)
		}
		authors = strings.Join(users, ", ")
	}
	text := strings.Split(strings.TrimSpace(summary), "\n")
	for k := range text {
		text[k] = w.linkify(strings.TrimRightFunc(text[k], unicode.IsSpace))
	}
	continuation := ""
	if len(text) > 1 {
		continuation = "  " + strings.Join(text[1:], "\n  ")
	}
	if w.format.Template != "" {
		line, err := w.template(text[0], pullLink, commitLink, authors)
		if err != nil {
			return "", err
		}
		return line + "\n" + continuation, nil
	}
	parts := []string{pullLink, commitLink}
	if authors != "" {
		parts = append(parts, "Thanks "+authors+"!")
	}
	parts = slices.DeleteFunc(parts, func(part string) bool { return part == "" })
	prefix := ""
	if len(parts) > 0 {
		prefix = " " + strings.Join(parts, " ") + " -"
	}
	return "\n\n-" + prefix + " " + text[0] + "\n" + continuation, nil
}

// template returns the template of the format with its tokens replaced: {summary} by summary,
// {ref} by the pull request or else the commit in parentheses, {pull}, {commit} and {authors} by
// their links, and the spaces at the end removed. It returns an error that wraps [ErrChangelog]
// for a token that it does not define.
func (w *renderer) template(summary, pull, commit, authors string) (string, error) {
	ref := ""
	if link := cmp.Or(pull, commit); link != "" {
		ref = "(" + link + ")"
	}
	tokens := map[string]string{"summary": summary, "ref": ref, "pull": pull, "commit": commit, "authors": authors}
	var unknown string
	line := templateToken.ReplaceAllStringFunc(w.format.Template, func(match string) string {
		value, ok := tokens[match[1:len(match)-1]]
		if !ok && unknown == "" {
			unknown = match
		}
		return value
	})
	if unknown != "" {
		return "", fmt.Errorf("%w: the token %s of the template, which is none of {summary}, {ref}, {pull}, "+
			"{commit} and {authors}", ErrChangelog, unknown)
	}
	return strings.TrimRightFunc(line, unicode.IsSpace), nil
}

// dependencyLine returns the line that lists updated, the requirements that a release rewrites,
// after the commits of sets, the changesets of the packages that they require, in the format of w.
// It returns the empty string for no update.
func (w *renderer) dependencyLine(ctx context.Context, sets []*changeset.Changeset, updated []update) (string, error) {
	if w.format.Format == ChangelogGitHub {
		if err := w.checkRepo(); err != nil {
			return "", err
		}
	}
	if len(updated) == 0 {
		return "", nil
	}
	var out []string
	if w.format.Format == ChangelogGitHub {
		var links []string
		for _, s := range sets {
			commit := w.commits[s.ID]
			if commit == "" {
				continue
			}
			link, _, _, err := w.host.Forge.CommitLinks(ctx, w.repo, commit)
			if err != nil {
				return "", err
			}
			links = append(links, cmp.Or(link, "`"+short(commit)+"`"))
		}
		out = append(out, "- Updated dependencies ["+strings.Join(links, ", ")+"]:")
	} else {
		for _, s := range sets {
			line := "- Updated dependencies"
			if commit := w.commits[s.ID]; commit != "" {
				line += " [" + short(commit) + "]"
			}
			out = append(out, line)
		}
	}
	for _, u := range updated {
		out = append(out, "  - "+u.name+"@"+u.version.String())
	}
	return strings.Join(out, "\n"), nil
}

// checkRepo returns an error that wraps [ErrChangelog] for a repository of w that is not
// owner/name, and nil otherwise.
func (w *renderer) checkRepo() error {
	if !repoPattern.MatchString(w.repo) {
		return fmt.Errorf("%w: the repository %q, which is not owner/name: name it in the options of the "+
			"changelog or in GITHUB_REPOSITORY", ErrChangelog, w.repo)
	}
	return nil
}

// linkify returns line with each reference to an issue, such as #123, linked to the issue, and
// with each Markdown link kept as it is.
func (w *renderer) linkify(line string) string {
	return issueRef.ReplaceAllStringFunc(line, func(match string) string {
		if !strings.HasPrefix(match, "#") {
			return match
		}
		return fmt.Sprintf("[%s](%s/%s/issues/%s)", match, w.server, w.repo, match[1:])
	})
}

// rewrites reports whether a requirement of the section kind takes a version that it resolves as
// resolution, for a release at bump: always for a requirement that excludes or pins the version,
// and for one that selects it when bump is at least updateInternalDependencies, or for a peer
// requirement unless onlyUpdatePeerDependentsWhenOutOfRange is set.
func (c *Config) rewrites(resolution language.Resolution, kind workspace.Kind, bump version.Bump) bool {
	if resolution != language.ResolutionSelected {
		return true
	}
	if kind == workspace.KindPeer {
		return !c.OnlyUpdatePeerDependentsWhenOutOfRange
	}
	return bump.AtLeast(c.UpdateInternalDependencies)
}

// dependencyUpdates returns the requirements of the package of r on other packages that the entry
// of r lists, and the changesets of the releases of those packages, in the order of the plan. A
// requirement counts as the runtime requirement, or the peer requirement of a package without one,
// as changesets reads dependencies and peerDependencies. It counts when [Config.rewrites] holds for
// the release of its package, and, for a package outside the plan, when the requirement pins the
// current version of the package.
func dependencyUpdates(g *Graph, c *Config, plan *Plan, r *Release) ([]update, []*changeset.Changeset) {
	i := g.index[r.Name]
	versioner := g.roles[i].versioner
	var updated []update
	ids := map[string]bool{}
	planned := map[int]bool{}
	for k := range plan.Releases {
		rel := &plan.Releases[k]
		j := g.index[rel.Name]
		planned[j] = true
		req, kind, ok := shippedRequirement(g, i, j)
		if !ok {
			continue
		}
		resolution, err := versioner.Resolve(req, rel.New)
		if err != nil || !c.rewrites(resolution, kind, rel.Bump) {
			continue
		}
		updated = append(updated, update{name: g.pkgs[j].Name, version: rel.New})
		for _, id := range rel.Changesets {
			ids[id] = true
		}
	}
	for j := range g.pkgs {
		if planned[j] {
			continue
		}
		req, _, ok := shippedRequirement(g, i, j)
		if !ok {
			continue
		}
		resolution, err := versioner.Resolve(req, g.pkgs[j].Version)
		if err == nil && resolution == language.ResolutionPinned {
			updated = append(updated, update{name: g.pkgs[j].Name, version: g.pkgs[j].Version})
		}
	}
	var sets []*changeset.Changeset
	for k := range plan.Changesets {
		if ids[plan.Changesets[k].ID] {
			sets = append(sets, &plan.Changesets[k])
		}
	}
	return updated, sets
}

// shippedRequirement returns the requirement of the package at index i on the package at index j
// of its toolchain that an entry reads, with its section: the runtime requirement, or the peer
// requirement of a package without one. It reports false for a package that i does not require in
// either section.
func shippedRequirement(g *Graph, i, j int) (string, workspace.Kind, bool) {
	if i == j || g.pkgs[i].Toolchain != g.pkgs[j].Toolchain {
		return "", "", false
	}
	for _, kind := range []workspace.Kind{workspace.KindRuntime, workspace.KindPeer} {
		for _, d := range g.pkgs[i].Deps {
			if d.Kind == kind && d.Name == g.pkgs[j].Name {
				return d.Req, kind, true
			}
		}
	}
	return "", "", false
}

// renderSection returns the section of level, such as ### Minor Changes, with each of lines trimmed
// and between one and two line breaks before each line, as changesets renders a section. It returns
// the empty string for lines without a line that is not empty.
func renderSection(level version.Bump, lines []string) string {
	var b strings.Builder
	breaks := 2
	for _, line := range lines {
		if line == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString("### " + strings.ToUpper(string(level[:1])) + string(level[1:]) + " Changes")
		}
		breaks += len(line) - len(strings.TrimLeft(line, "\n"))
		b.WriteString(strings.Repeat("\n", min(max(breaks, 1), 2)))
		b.WriteString(strings.TrimSpace(line))
		breaks = len(line) - len(strings.TrimRight(line, "\n"))
	}
	return b.String()
}

// updateChangelog returns data, the content of the changelog of the package name, with entry
// inserted as changesets inserts it: in a new changelog under the heading # name, before the first
// heading of a version, or after the first line of a changelog without one.
func updateChangelog(data []byte, name, entry string) []byte {
	entry = strings.TrimSpace(entry)
	text := string(data)
	if text == "" {
		return []byte("# " + name + "\n\n" + entry + "\n")
	}
	if at := versionHeading.FindStringIndex(text); at != nil {
		return []byte(text[:at[0]] + entry + "\n\n" + text[at[0]:])
	}
	if first, rest, ok := strings.Cut(text, "\n"); ok {
		return []byte(first + "\n\n" + entry + "\n" + rest)
	}
	return []byte(text + "\n\n" + entry + "\n")
}

// cut returns s without the first match of pattern, and the first group of that match, or s and
// the empty string for s without a match.
func cut(pattern *regexp.Regexp, s string) (string, string) {
	m := pattern.FindStringSubmatchIndex(s)
	if m == nil {
		return s, ""
	}
	return s[:m[0]] + s[m[1]:], s[m[2]:m[3]]
}

// short returns the first seven characters of commit, as changesets abbreviates a commit.
func short(commit string) string {
	return commit[:min(7, len(commit))]
}
