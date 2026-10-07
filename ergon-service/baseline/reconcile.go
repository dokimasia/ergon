// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/baseline/render"
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
	// the lock: the baseline, the answers, an option or the local file changed. [Repository.Sync]
	// rewrites it.
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
// leaves because they conflict, and the entries of files of the lock that follows the change.
type plan struct {
	// writes are the files to write, sorted by path.
	writes []write

	// removals are the files to remove, sorted by path.
	removals []string

	// conflicts are the managed files that the plan leaves unchanged, because they were edited by
	// hand or exist with other content, sorted by path.
	conflicts []string

	// files are the entries of files of the lock after the change, sorted by path.
	files []lock.File
}

// reconcile returns the plan that brings the repository to rend. recorded are the entries of files
// of the previous lock, and nil for a repository without one.
//
// A managed file is written when it is missing, and when the rendering differs from a file that
// is unedited since the lock. A file that equals the rendering is kept. A file that differs from the
// rendering and from the lock conflicts, unless opts.Force overwrites it. A file that the lock
// records and the rendering no longer has is removed when it is unedited, and conflicts otherwise
// unless opts.Force removes it. A seeded file is written when it is missing. .ergon.yaml is
// written when its sections change a key or a value. A conflicting file keeps its entry in the
// lock.
func (r *Repository) reconcile(recorded []lock.File, rend *rendering, opts Options) (plan, error) {
	var p plan
	previous := lock.Lock{Files: recorded}
	for _, t := range rend.files {
		existing, ok, err := r.read(t.Path)
		if err != nil {
			return plan{}, err
		}
		if t.Class == render.Seeded {
			if !ok {
				p.writes = append(p.writes, write{path: t.Path, content: t.Content})
			}
			continue
		}
		entry, known := previous.File(t.Path)
		current := lock.File{Path: t.Path, Producer: t.Producer, Local: t.local, SHA256: lock.Digest(t.Content)}
		if ok && bytes.Equal(existing, t.Content) {
			p.files = append(p.files, current)
		} else if !ok || (known && lock.Digest(existing) == entry.SHA256) || opts.Force {
			p.writes = append(p.writes, write{path: t.Path, content: t.Content})
			p.files = append(p.files, current)
		} else {
			p.conflicts = append(p.conflicts, t.Path)
			if known {
				p.files = append(p.files, entry)
			}
		}
	}
	if rend.configured {
		p.writes = append(p.writes, write{path: language.Config, content: rend.config})
	}
	for _, entry := range recorded {
		rendered := func(t target) bool { return t.Path == entry.Path && t.Class == render.Managed }
		if slices.ContainsFunc(rend.files, rendered) {
			continue
		}
		existing, ok, err := r.read(entry.Path)
		if err != nil {
			return plan{}, err
		}
		if !ok {
			continue
		}
		if lock.Digest(existing) == entry.SHA256 || opts.Force {
			p.removals = append(p.removals, entry.Path)
			continue
		}
		p.conflicts = append(p.conflicts, entry.Path)
		p.files = append(p.files, entry)
	}
	slices.SortFunc(p.writes, func(a, b write) int { return strings.Compare(a.path, b.path) })
	slices.SortFunc(p.files, func(a, b lock.File) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(p.conflicts)
	return p, nil
}

// inspect returns the findings of the managed files: the targets of a rendering, compared with
// the repository and with the entries recorded in the lock, sorted by path.
func (r *Repository) inspect(recorded []lock.File, targets []target) ([]Finding, error) {
	var findings []Finding
	previous := lock.Lock{Files: recorded}
	for _, t := range targets {
		if t.Class != render.Managed {
			continue
		}
		existing, ok, err := r.read(t.Path)
		if err != nil {
			return nil, err
		}
		entry, known := previous.File(t.Path)
		if !ok {
			findings = append(findings, Finding{Path: t.Path, Problem: Missing})
		} else if bytes.Equal(existing, t.Content) {
			if !known || entry.SHA256 != lock.Digest(t.Content) {
				findings = append(findings, Finding{Path: t.Path, Problem: Outdated})
			}
		} else if known && lock.Digest(existing) == entry.SHA256 {
			findings = append(findings, Finding{Path: t.Path, Problem: Outdated})
		} else {
			findings = append(findings, Finding{Path: t.Path, Problem: Edited})
		}
	}
	for _, entry := range recorded {
		rendered := func(t target) bool { return t.Path == entry.Path && t.Class == render.Managed }
		if slices.ContainsFunc(targets, rendered) {
			continue
		}
		existing, ok, err := r.read(entry.Path)
		if err != nil {
			return nil, err
		}
		problem := Outdated
		if ok && lock.Digest(existing) != entry.SHA256 {
			problem = Edited
		}
		findings = append(findings, Finding{Path: entry.Path, Problem: problem})
	}
	slices.SortFunc(findings, func(a, b Finding) int { return strings.Compare(a.Path, b.Path) })
	return findings, nil
}
