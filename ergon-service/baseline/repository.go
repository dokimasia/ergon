// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// The errors of the commands.
var (
	// ErrInvalidOpen is the error of [Open] for a missing file system, catalog or version, and for
	// an invalid base producer.
	ErrInvalidOpen = errors.New("baseline: invalid arguments to Open")

	// ErrInitialized is the error of [Repository.New] for a repository that has a lock.
	ErrInitialized = errors.New("baseline: the repository has a lock, so use add or sync")

	// ErrNotInitialized is the error of every other command for a repository without a lock.
	ErrNotInitialized = errors.New("baseline: the repository has no lock, so run new first")

	// ErrUnknownLanguage is the error for a language that the catalog does not have.
	ErrUnknownLanguage = errors.New("baseline: unknown language")

	// ErrLanguagePresent is the error of [Repository.Add] for a language that the answers already
	// have, and of every command for a language that the answers name twice.
	ErrLanguagePresent = errors.New("baseline: language already present")

	// ErrLanguageAbsent is the error of [Repository.Remove] for a language that the answers do not
	// have.
	ErrLanguageAbsent = errors.New("baseline: language not present")

	// ErrUnsupported is the error for a language that registered no [language.Initializer].
	ErrUnsupported = errors.New("baseline: language without an init role")

	// ErrConflict is the error for managed files that a command leaves, because they were edited
	// by hand or exist with other content. Its text lists the paths.
	ErrConflict = errors.New("baseline: managed files edited by hand or existing with other content")
)

// Options are the options of the commands that write.
type Options struct {
	// Force overwrites a managed file that was edited by hand or exists with other content, and
	// removes an edited file that the rendering no longer has.
	Force bool
}

// Action is what a command did to a file.
type Action string

const (
	// Wrote is a file that a command created or rewrote.
	Wrote Action = "wrote"

	// Removed is a file that a command removed.
	Removed Action = "removed"
)

// Change is a file that a command wrote or removed.
type Change struct {
	// Path is the path of the file in the repository.
	Path string

	// Action is what the command did to the file.
	Action Action
}

// Repository is a repository that ergon init sets up, on a file system that confines every path
// to its directory. Its producers are the base producers of [Open], then the
// [language.Initializer] of each language of the answers, in the order of the catalog. A
// toolchain with an Initializer renders the files that its languages share, once, before its
// first language of the answers.
//
// # Concurrency
//
// A Repository is not safe for concurrent use, and two processes must not run commands on one
// repository at a time: each command reads the files, plans, and then writes.
type Repository struct {
	// fsys is the file system of the repository.
	fsys FS

	// catalog has the languages and their roles.
	catalog *language.Catalog

	// version is the version of ergon that the lock records.
	version string

	// base are the producers before the languages, such as the common files and the GitHub files.
	base []Producer
}

// Open returns the repository on fsys, whose toolchains, languages and roles are in catalog.
// version is the version of ergon that the lock records. base are the producers that run before
// the languages, in order. Open returns an error that wraps [ErrInvalidOpen] for a nil fsys or
// catalog, an empty version, and a base producer without an initializer, whose name is not a
// valid name, or whose name is the name of another producer or of a toolchain or a language of
// catalog.
func Open(fsys FS, catalog *language.Catalog, version string, base ...Producer) (*Repository, error) {
	if fsys == nil || catalog == nil || version == "" {
		return nil, fmt.Errorf("%w: a file system, a catalog and a version are required", ErrInvalidOpen)
	}
	for i, p := range base {
		_, isLanguage := catalog.Language(workspace.Language(p.Name))
		_, isToolchain := catalog.Toolchain(workspace.Toolchain(p.Name))
		taken := slices.ContainsFunc(base[:i], func(q Producer) bool { return q.Name == p.Name })
		if p.Initializer == nil || !workspace.Language(p.Name).Valid() || isLanguage || isToolchain || taken {
			return nil, fmt.Errorf("%w: producer %q", ErrInvalidOpen, p.Name)
		}
	}
	return &Repository{fsys: fsys, catalog: catalog, version: version, base: slices.Clone(base)}, nil
}

// New writes the files of a repository without a lock: the managed files, the seeded files that
// are missing, the keys of the configured files, and the lock. It returns the files it wrote.
//
// It returns an error that wraps [ErrInitialized] for a repository with a lock, [ErrInvalidLock]
// for a lock that does not parse, the error of a producer for answers that it cannot render, and
// an error that wraps [ErrConflict] for a managed file that exists with other content, unless
// opts.Force overwrites it. New writes nothing when it returns an error before its first write. It
// reads a and does not modify it.
func (r *Repository) New(a *language.Answers, opts Options) ([]Change, error) {
	data, ok, err := r.read(lockPath)
	if err != nil {
		return nil, err
	}
	if ok {
		if _, err := decodeLock(data); err != nil {
			return nil, err
		}
		return nil, ErrInitialized
	}
	return r.change(nil, a, opts)
}

// Add adds languages to the answers of the lock, and writes their files: the files of each
// language, its fragments of the shared files, and the lock. It returns the files it wrote.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock,
// [ErrLanguagePresent] for a language that the answers already have, and [ErrConflict] for a
// managed file that it would change and that was edited by hand, unless opts.Force overwrites
// it. Add writes nothing when it returns an error before its first write.
func (r *Repository) Add(languages []workspace.Language, opts Options) ([]Change, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	for _, name := range languages {
		if slices.Contains(l.Answers.Languages, name) {
			return nil, fmt.Errorf("%w: %s", ErrLanguagePresent, name)
		}
	}
	a := l.Answers
	a.Languages = slices.Concat(a.Languages, languages)
	return r.change(l.Files, &a, opts)
}

// Remove removes languages from the answers of the lock, removes the files of each language, and
// rewrites the shared files and the lock. It returns the files it wrote and removed.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock,
// [ErrUnknownLanguage] for a language that the catalog does not have, [ErrLanguageAbsent] for a
// language that the answers do not have, and [ErrConflict] for a managed file that it would change
// or remove and that was edited by hand. Remove writes nothing when it returns an error before its
// first write.
func (r *Repository) Remove(languages []workspace.Language) ([]Change, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	remaining := slices.Clone(l.Answers.Languages)
	for _, name := range languages {
		if err := r.known(name); err != nil {
			return nil, err
		}
		i := slices.Index(remaining, name)
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrLanguageAbsent, name)
		}
		remaining = slices.Delete(remaining, i, i+1)
	}
	a := l.Answers
	a.Languages = remaining
	return r.change(l.Files, &a, Options{})
}

// Sync brings the files to the rendering of the installed ergon, with the answers of the lock
// changed by update: it writes the missing and outdated managed files, the missing seeded files,
// the keys of the configured files, and the lock. update may be nil. It returns the files it
// wrote and removed.
//
// A managed file that was edited by hand conflicts, unless opts.Force overwrites it. Sync writes
// every other file, keeps the conflicting file and its entry in the lock, and then returns an
// error that wraps [ErrConflict]. It returns an error that wraps [ErrNotInitialized] for a
// repository without a lock, and the error of a producer for answers that it cannot render.
func (r *Repository) Sync(update func(*language.Answers), opts Options) ([]Change, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	a := l.Answers
	a.Languages = slices.Clone(a.Languages)
	if update != nil {
		update(&a)
	}
	targets, ordered, err := r.targets(&a)
	if err != nil {
		return nil, err
	}
	p, err := r.reconcile(l.Files, targets, opts)
	if err != nil {
		return nil, err
	}
	changes, err := r.apply(&p, &ordered)
	if err != nil || len(p.conflicts) == 0 {
		return changes, err
	}
	return changes, fmt.Errorf("%w: %s", ErrConflict, strings.Join(p.conflicts, ", "))
}

// Check returns the managed files that differ from the rendering of the installed ergon for the
// answers of the lock, sorted by path, and writes nothing. It returns an error that wraps
// [ErrNotInitialized] for a repository without a lock.
func (r *Repository) Check() ([]Finding, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	targets, _, err := r.targets(&l.Answers)
	if err != nil {
		return nil, err
	}
	return r.inspect(l.Files, targets)
}

// change renders a, plans against the recorded lock entries, and applies the plan when nothing
// conflicts. It returns an error that wraps [ErrConflict], and writes nothing, when a managed file
// conflicts. It reads a and does not modify it.
func (r *Repository) change(recorded []lockFile, a *language.Answers, opts Options) ([]Change, error) {
	targets, ordered, err := r.targets(a)
	if err != nil {
		return nil, err
	}
	p, err := r.reconcile(recorded, targets, opts)
	if err != nil {
		return nil, err
	}
	if len(p.conflicts) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrConflict, strings.Join(p.conflicts, ", "))
	}
	return r.apply(&p, &ordered)
}

// targets returns the rendering of a, with the local files merged into the managed files, and a
// copy of a with its languages in the order of the catalog. It reads a and does not modify it.
// The initializer of a toolchain renders before the first language of the toolchain. It returns
// an error that wraps [ErrUnknownLanguage] for a language that the catalog does not have,
// [ErrLanguagePresent] for a language that a names twice, [ErrUnsupported] for a language without
// an initializer, and the error of a producer or of a local file.
func (r *Repository) targets(a *language.Answers) ([]target, language.Answers, error) {
	ordered := *a
	producers := slices.Clone(r.base)
	for i, name := range a.Languages {
		if err := r.known(name); err != nil {
			return nil, ordered, err
		}
		if slices.Contains(a.Languages[:i], name) {
			return nil, ordered, fmt.Errorf("%w: %s, which the answers name twice", ErrLanguagePresent, name)
		}
	}
	languages := make([]workspace.Language, 0, len(a.Languages))
	var toolchains []workspace.Toolchain
	for d := range r.catalog.Languages() {
		if !slices.Contains(a.Languages, d.Name) {
			continue
		}
		initializer, ok := language.Role[language.Initializer](r.catalog, d.Name)
		if !ok {
			return nil, ordered, fmt.Errorf("%w: %s", ErrUnsupported, d.Name)
		}
		if !slices.Contains(toolchains, d.Toolchain) {
			toolchains = append(toolchains, d.Toolchain)
			if shared, ok := language.ToolchainRole[language.Initializer](r.catalog, d.Toolchain); ok {
				producers = append(producers, Producer{Name: string(d.Toolchain), Initializer: shared})
			}
		}
		languages = append(languages, d.Name)
		producers = append(producers, Producer{Name: string(d.Name), Initializer: initializer})
	}
	ordered.Languages = languages
	targets, err := render(producers, &ordered)
	if err != nil {
		return nil, ordered, err
	}
	if err := r.applyLocal(targets); err != nil {
		return nil, ordered, err
	}
	return targets, ordered, nil
}

// known returns nil for a language of the catalog, and for any other name an error that wraps
// [ErrUnknownLanguage] and lists the languages of the catalog in their order.
func (r *Repository) known(name workspace.Language) error {
	if _, ok := r.catalog.Language(name); ok {
		return nil
	}
	var names []string
	for d := range r.catalog.Languages() {
		names = append(names, string(d.Name))
	}
	return fmt.Errorf("%w: %q, which is none of %s", ErrUnknownLanguage, name, strings.Join(names, ", "))
}

// apply writes and removes the files of p, then writes the lock of a and p's file entries when it
// differs from the lock in the repository. It returns the files that it wrote and removed, those
// before a failure included, and the error of the first write or removal that fails.
func (r *Repository) apply(p *plan, a *language.Answers) ([]Change, error) {
	var changes []Change
	for _, w := range p.writes {
		if err := r.write(w.path, w.content); err != nil {
			return changes, err
		}
		changes = append(changes, Change{Path: w.path, Action: Wrote})
	}
	for _, name := range p.removals {
		if err := r.remove(name); err != nil {
			return changes, err
		}
		changes = append(changes, Change{Path: name, Action: Removed})
	}
	files := p.files
	if files == nil {
		files = []lockFile{}
	}
	next := lock{Ergon: r.version, Answers: *a, Files: files}
	encoded := next.encode()
	previous, _, err := r.read(lockPath)
	if err != nil || bytes.Equal(previous, encoded) {
		return changes, err
	}
	if err := r.write(lockPath, encoded); err != nil {
		return changes, err
	}
	return append(changes, Change{Path: lockPath, Action: Wrote}), nil
}

// readLock returns the lock of the repository. It returns an error that wraps
// [ErrNotInitialized] for a repository without one, and [ErrInvalidLock] for a lock that does
// not parse.
func (r *Repository) readLock() (lock, error) {
	data, ok, err := r.read(lockPath)
	if err != nil {
		return lock{}, err
	}
	if !ok {
		return lock{}, ErrNotInitialized
	}
	return decodeLock(data)
}
