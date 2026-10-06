// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
)

// Problem is what [Repository.Check] finds wrong with a managed file.
type Problem string

const (
	// Missing is a managed file that the repository does not have.
	Missing Problem = "missing"

	// Edited is a managed file whose content differs from the lock and from the rendering, as a
	// change by hand leaves it.
	Edited Problem = "edited"

	// Outdated is a managed file that is unedited since the lock, while the rendering differs from
	// the lock: the baseline, the answers or the local file changed. [Repository.Sync] rewrites it.
	Outdated Problem = "outdated"
)

// Finding is a managed file that differs from the baseline.
type Finding struct {
	// Path is the path of the file in the repository.
	Path string `json:"path"`

	// Problem is what differs.
	Problem Problem `json:"problem"`
}

// write is a file that a plan writes.
type write struct {
	// path is the path of the file in the repository.
	path string

	// content is the new content of the file.
	content []byte
}

// plan is the change of a command: the files to write and to remove, the managed files that it
// leaves because they conflict, and the file entries of the lock that follows the change.
type plan struct {
	// writes are the files to write, sorted by path.
	writes []write

	// removals are the files to remove, sorted by path.
	removals []string

	// conflicts are the managed files that the plan leaves unchanged, because they were edited by
	// hand or exist with other content, sorted by path.
	conflicts []string

	// files are the file entries of the lock after the change, sorted by path.
	files []lockFile
}

// reconcile returns the plan that brings the repository to the targets of a rendering. recorded
// are the file entries of the previous lock, and nil for a repository without one.
//
// A managed file is written when it is missing, and when the rendering differs from a file that
// is unedited since the lock. A file that equals the rendering is kept. A file that differs from the
// rendering and from the lock conflicts, unless opts.Force overwrites it. A file that the lock
// records and the rendering no longer has is removed when it is unedited, and conflicts otherwise
// unless opts.Force removes it. A seeded file is written when it is missing. A configured file
// receives the keys of its rendering. A conflicting file keeps its entry in the lock.
func (r *Repository) reconcile(recorded []lockFile, targets []target, opts Options) (plan, error) {
	var p plan
	for _, t := range targets {
		existing, ok, err := r.read(t.path)
		if err != nil {
			return plan{}, err
		}
		if t.class == language.Seeded {
			if !ok {
				p.writes = append(p.writes, write{path: t.path, content: t.content})
			}
			continue
		}
		if t.class == language.Configured {
			content, changed, err := configure(&t, existing, ok)
			if err != nil {
				return plan{}, err
			}
			if changed {
				p.writes = append(p.writes, write{path: t.path, content: content})
			}
			continue
		}
		entry, known := find(recorded, t.path)
		current := lockFile{Path: t.path, Producer: t.producer, Local: t.local, SHA256: digest(t.content)}
		if ok && bytes.Equal(existing, t.content) {
			p.files = append(p.files, current)
		} else if !ok || (known && digest(existing) == entry.SHA256) || opts.Force {
			p.writes = append(p.writes, write{path: t.path, content: t.content})
			p.files = append(p.files, current)
		} else {
			p.conflicts = append(p.conflicts, t.path)
			if known {
				p.files = append(p.files, entry)
			}
		}
	}
	for _, entry := range recorded {
		rendered := func(t target) bool { return t.path == entry.Path && t.class == language.Managed }
		if slices.ContainsFunc(targets, rendered) {
			continue
		}
		existing, ok, err := r.read(entry.Path)
		if err != nil {
			return plan{}, err
		}
		if !ok {
			continue
		}
		if digest(existing) == entry.SHA256 || opts.Force {
			p.removals = append(p.removals, entry.Path)
			continue
		}
		p.conflicts = append(p.conflicts, entry.Path)
		p.files = append(p.files, entry)
	}
	slices.SortFunc(p.files, func(a, b lockFile) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(p.conflicts)
	return p, nil
}

// inspect returns the findings of the managed files: the targets of a rendering, compared with
// the repository and with the entries recorded in the lock, sorted by path.
func (r *Repository) inspect(recorded []lockFile, targets []target) ([]Finding, error) {
	var findings []Finding
	for _, t := range targets {
		if t.class != language.Managed {
			continue
		}
		existing, ok, err := r.read(t.path)
		if err != nil {
			return nil, err
		}
		entry, known := find(recorded, t.path)
		if !ok {
			findings = append(findings, Finding{Path: t.path, Problem: Missing})
		} else if bytes.Equal(existing, t.content) {
			if !known || entry.SHA256 != digest(t.content) {
				findings = append(findings, Finding{Path: t.path, Problem: Outdated})
			}
		} else if known && digest(existing) == entry.SHA256 {
			findings = append(findings, Finding{Path: t.path, Problem: Outdated})
		} else {
			findings = append(findings, Finding{Path: t.path, Problem: Edited})
		}
	}
	for _, entry := range recorded {
		rendered := func(t target) bool { return t.path == entry.Path && t.class == language.Managed }
		if slices.ContainsFunc(targets, rendered) {
			continue
		}
		existing, ok, err := r.read(entry.Path)
		if err != nil {
			return nil, err
		}
		problem := Outdated
		if ok && digest(existing) != entry.SHA256 {
			problem = Edited
		}
		findings = append(findings, Finding{Path: entry.Path, Problem: problem})
	}
	slices.SortFunc(findings, func(a, b Finding) int { return strings.Compare(a.Path, b.Path) })
	return findings, nil
}

// find returns the entry of path in files, and reports whether files has one.
func find(files []lockFile, path string) (lockFile, bool) {
	i := slices.IndexFunc(files, func(f lockFile) bool { return f.Path == path })
	if i < 0 {
		return lockFile{}, false
	}
	return files[i], true
}
