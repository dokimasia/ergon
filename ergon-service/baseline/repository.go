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
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.dokimi.dev/ergon/service/baseline/render"
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

	// ErrUnsupported is the error for a language that registered no [language.Producer].
	ErrUnsupported = errors.New("baseline: language without a producer")

	// ErrConflict is the error for managed files that a command leaves, because they were edited
	// by hand or exist with other content. Its text lists the paths.
	ErrConflict = errors.New("baseline: managed files edited by hand or existing with other content")

	// ErrInvalidFile is the error for a producer that renders .ergon.yaml, which ergon init writes
	// from the options of the producers alone. It is a defect of the producer.
	ErrInvalidFile = errors.New("baseline: a producer renders .ergon.yaml")

	// ErrUnknownSection is the error of [Repository.Options] for a section that no producer of the
	// repository with options has.
	ErrUnknownSection = errors.New("baseline: no producer of the repository has the section")

	// ErrNewerLock is the error of every command but New for a lock that a newer release of ergon
	// wrote than the running one. Its text contains both releases.
	ErrNewerLock = errors.New("baseline: the lock is of a newer release of ergon")

	// ErrDevelopmentBuild is the error of every command that writes a lock, for a build of ergon
	// without a release that would write a new lock or a lock that a release wrote. The lock would
	// name a version that no release has, so a CI job could not install it.
	ErrDevelopmentBuild = errors.New("baseline: a build of ergon without a release writes no lock of a release")
)

// ConflictError is the error of a command for the managed files that it leaves, because they were
// edited by hand or exist with other content. It wraps [ErrConflict], and every error of a command
// that wraps ErrConflict is one.
type ConflictError struct {
	// Paths are the paths of the managed files that the command left: the files that it would write,
	// in the order of their paths, and then the files that it would remove.
	Paths []string
}

// Error returns the text of ErrConflict and the paths, separated by commas.
func (e *ConflictError) Error() string {
	return ErrConflict.Error() + ": " + strings.Join(e.Paths, ", ")
}

// Unwrap returns ErrConflict.
func (*ConflictError) Unwrap() error {
	return ErrConflict
}

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

// Producer is a base producer of a repository, which renders before the languages, such as the
// producer of the common files.
type Producer struct {
	// Producer renders the files of the producer.
	Producer language.Producer

	// Name identifies the producer in the lock, and names its section of .ergon.yaml, such as
	// common.
	Name string
}

// Repository is a repository that ergon init sets up, on a file system that confines every path
// to its directory. Its producers are the base producers of [Open], then the producer of each
// language of the answers, in the order of the catalog. A toolchain with a producer renders the
// files that its languages share, once, before its first language of the answers.
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

// rendering is what a command renders: the files of the producers, with each local file merged
// into its managed file, and .ergon.yaml with the sections of the producers.
type rendering struct {
	// files are the files of the producers, sorted by path.
	files []target

	// config is .ergon.yaml with the sections of the producers, and nil when no producer has
	// options and the repository has no section to drop.
	config []byte

	// configured reports that config differs from .ergon.yaml in a key or a value.
	configured bool
}

// target is a file of a rendering.
type target struct {
	// local is the digest of the local file that the content contains, or empty.
	local string

	render.File
}

// Open returns the repository on fsys, whose toolchains, languages and roles are in catalog.
// version is the version of ergon that the lock records. base are the producers that run before
// the languages, in order. Open returns an error that wraps [ErrInvalidOpen] for a nil fsys or
// catalog, an empty version, and a base producer without a producer, whose name is not a valid
// name, or whose name is the name of another producer or of a toolchain or a language of catalog.
func Open(fsys FS, catalog *language.Catalog, version string, base ...Producer) (*Repository, error) {
	if fsys == nil || catalog == nil || version == "" {
		return nil, fmt.Errorf("%w: a file system, a catalog and a version are required", ErrInvalidOpen)
	}
	for i, p := range base {
		_, isLanguage := catalog.Language(workspace.Language(p.Name))
		_, isToolchain := catalog.Toolchain(workspace.Toolchain(p.Name))
		taken := slices.ContainsFunc(base[:i], func(q Producer) bool { return q.Name == p.Name })
		if p.Producer == nil || !workspace.Language(p.Name).Valid() || isLanguage || isToolchain || taken {
			return nil, fmt.Errorf("%w: producer %q", ErrInvalidOpen, p.Name)
		}
	}
	return &Repository{fsys: fsys, catalog: catalog, version: version, base: slices.Clone(base)}, nil
}

// New writes the files of a repository without a lock: the managed files, the seeded files that
// are missing, the section of each producer that has options in .ergon.yaml, and the lock. It
// writes each answer of a into the key of .ergon.yaml that states it, whatever value the key had.
// It returns the files it wrote.
//
// It returns an error that wraps [ErrInitialized] for a repository with a lock,
// [lock.ErrInvalid] for a lock that does not parse, [language.ErrInvalidAnswer] for answers that no
// producer can render, [options.ErrInvalid] for an existing .ergon.yaml that the producers do not
// accept, the error of a producer, and an error that wraps [ErrConflict] for a managed file that
// exists with other content, unless opts.Force overwrites it. It returns an error that wraps
// [ErrDevelopmentBuild] for a build of ergon without a release, before it reads the lock. New writes
// nothing when it returns an error before its first write. It reads a and does not modify it.
func (r *Repository) New(a *language.Answers, opts Options) ([]Change, error) {
	if err := r.writable(nil); err != nil {
		return nil, err
	}
	data, ok, err := r.read(lock.Path)
	if err != nil {
		return nil, err
	}
	if ok {
		if _, err := lock.Decode(data); err != nil {
			return nil, err
		}
		return nil, ErrInitialized
	}
	return r.change(nil, a, opts)
}

// Add adds languages to the answers of the lock, and writes their files: the files of each
// language, its fragments of the shared files, its section of .ergon.yaml, and the lock. It
// returns the files it wrote.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock,
// [ErrLanguagePresent] for a language that the answers already have, [options.ErrInvalid] for
// .ergon.yaml that the producers do not accept, and [ErrConflict] for a managed file that it would
// change and that was edited by hand, unless opts.Force overwrites it, and the errors of
// [Repository.writableLock]. Add writes nothing when it returns an error before its first write.
func (r *Repository) Add(languages []workspace.Language, opts Options) ([]Change, error) {
	l, err := r.writableLock()
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
	return r.change(&l, &a, opts)
}

// Remove removes languages from the answers of the lock, removes the files of each language and
// its section of .ergon.yaml, and rewrites the shared files and the lock. It returns the files it
// wrote and removed.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock,
// [ErrUnknownLanguage] for a language that the catalog does not have, [ErrLanguageAbsent] for a
// language that the answers do not have, [options.ErrInvalid] for .ergon.yaml that the producers do
// not accept, [ErrConflict] for a managed file that it would change or remove and that was edited by
// hand, and the errors of [Repository.writableLock]. Remove writes nothing when it returns an error
// before its first write.
func (r *Repository) Remove(languages []workspace.Language) ([]Change, error) {
	l, err := r.writableLock()
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
	return r.change(&l, &a, Options{})
}

// Sync brings the files to the rendering of the installed ergon, with the answers of the lock
// changed by update: it writes the missing and outdated managed files, the missing seeded files,
// the section of each producer that has options in .ergon.yaml, and the lock. update may be nil.
// It returns the files it wrote and removed.
//
// An option of .ergon.yaml whose value equals the value that the lock records takes the baseline
// value of the installed ergon. A key of .ergon.yaml that states an answer must have the answer of
// the lock or the answer that update sets, and Sync writes the answer that update sets into it.
//
// A managed file that was edited by hand conflicts, unless opts.Force overwrites it. Sync writes
// every other file, keeps the conflicting file and its entry in the lock, and then returns an error
// that wraps [ErrConflict]. It returns an error that wraps [ErrNotInitialized] for a repository
// without a lock, [language.ErrInvalidAnswer] for answers that no producer can render,
// [options.ErrInvalid] for .ergon.yaml that the producers do not accept or whose key of an answer
// has another value, the error of a producer, and the errors of [Repository.writableLock].
func (r *Repository) Sync(update func(*language.Answers), opts Options) ([]Change, error) {
	l, err := r.writableLock()
	if err != nil {
		return nil, err
	}
	a := l.Answers
	a.Languages = slices.Clone(a.Languages)
	if update != nil {
		update(&a)
	}
	rend, next, err := r.targets(&a, &l)
	if err != nil {
		return nil, err
	}
	p, err := r.reconcile(l.Files, &rend, opts)
	if err != nil {
		return nil, err
	}
	changes, err := r.apply(&p, &next)
	if err != nil || len(p.conflicts) == 0 {
		return changes, err
	}
	return changes, &ConflictError{Paths: p.conflicts}
}

// Check returns the managed files that differ from the rendering of the installed ergon for the
// answers of the lock and the options of .ergon.yaml, sorted by path, and writes nothing. It
// returns an error that wraps [ErrNotInitialized] for a repository without a lock, and
// [options.ErrInvalid] for .ergon.yaml that the producers do not accept.
func (r *Repository) Check() ([]Finding, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	rend, _, err := r.targets(&l.Answers, &l)
	if err != nil {
		return nil, err
	}
	return r.inspect(l.Files, rend.files)
}

// Options returns the options of the section name of .ergon.yaml, resolved as every command
// resolves them: from the section, the record of the lock and the baseline of the producer of that
// name among the producers of the repository. It does not write a file.
//
// It returns an error that wraps [ErrNotInitialized] for a repository without a lock,
// [ErrUnknownSection] for a name of no producer with options, the errors that [Repository.Check]
// returns for the answers of the lock, and [options.ErrInvalid] for .ergon.yaml that the producers
// do not accept.
func (r *Repository) Options(name string) (language.Options, error) {
	sections, err := r.Sections()
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(sections, func(s options.Section) bool { return s.Name == name })
	if i < 0 {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSection, name)
	}
	return sections[i].Options, nil
}

// Sections returns the options of each producer of the repository with options, in the order of
// the producers. It resolves each section as [Repository.Options] resolves one, and does not write
// a file. It returns the errors of Options, except ErrUnknownSection.
func (r *Repository) Sections() ([]options.Section, error) {
	l, err := r.readLock()
	if err != nil {
		return nil, err
	}
	units, next, err := r.units(&l.Answers)
	if err != nil {
		return nil, err
	}
	_, res, err := r.resolve(units, &next.Answers, &l)
	if err != nil {
		return nil, err
	}
	return res.Sections, nil
}

// change renders a with the options that previous records, plans against the file entries of
// previous, and applies the plan when nothing conflicts. previous is the lock of the repository,
// and nil for a repository without one. It returns an error that wraps [ErrConflict], and writes
// nothing, when a managed file conflicts. It reads a and does not modify it.
func (r *Repository) change(previous *lock.Lock, a *language.Answers, opts Options) ([]Change, error) {
	rend, next, err := r.targets(a, previous)
	if err != nil {
		return nil, err
	}
	var recorded []lock.File
	if previous != nil {
		recorded = previous.Files
	}
	p, err := r.reconcile(recorded, &rend, opts)
	if err != nil {
		return nil, err
	}
	if len(p.conflicts) > 0 {
		return nil, &ConflictError{Paths: p.conflicts}
	}
	return r.apply(&p, &next)
}

// targets returns the rendering of a, of the options of .ergon.yaml and of the local files, and
// the lock that follows the rendering without its entries of files. That lock has a copy of a with
// its languages in the order of the catalog, and the baseline value of each option. previous is the
// lock of the repository, whose record and answers resolve the options, and nil for a repository
// without one. targets reads a and previous and modifies neither.
//
// It returns an error that wraps [language.ErrInvalidAnswer] for answers that no producer can
// render, [ErrUnknownLanguage] for a language that the catalog does not have, [ErrLanguagePresent]
// for a language that a names twice, [ErrUnsupported] for a language without a producer,
// [options.ErrInvalid] for .ergon.yaml that the producers do not accept, [ErrInvalidFile] for a
// producer that renders .ergon.yaml, and the error of a producer, of the rendering or of a local
// file.
func (r *Repository) targets(a *language.Answers, previous *lock.Lock) (rendering, lock.Lock, error) {
	units, next, err := r.units(a)
	if err != nil {
		return rendering{}, lock.Lock{}, err
	}
	rend, err := r.render(units, &next.Answers, previous)
	if err != nil {
		return rendering{}, lock.Lock{}, err
	}
	next.Settings = rend.record
	return rend.rendering, next, nil
}

// units returns the producers of a, the base producers and then the toolchain and the producer of
// each language of a in the order of the catalog, and the lock that follows them without its
// entries of files and its record: a copy of a with its languages in the order of the catalog. It
// reads a and does not modify it. It returns the errors that [Repository.targets] states for the
// answers and the languages.
func (r *Repository) units(a *language.Answers) ([]render.Unit, lock.Lock, error) {
	if err := a.Validate(); err != nil {
		return nil, lock.Lock{}, err
	}
	for i, name := range a.Languages {
		if err := r.known(name); err != nil {
			return nil, lock.Lock{}, err
		}
		if slices.Contains(a.Languages[:i], name) {
			return nil, lock.Lock{}, fmt.Errorf("%w: %s, which the answers name twice", ErrLanguagePresent, name)
		}
	}
	next := lock.Lock{Ergon: r.version, Answers: *a}
	units := make([]render.Unit, 0, len(r.base)+2*len(a.Languages))
	for _, p := range r.base {
		units = append(units, render.Unit{Name: p.Name, Producer: p.Producer})
	}
	languages := make([]workspace.Language, 0, len(a.Languages))
	var toolchains []workspace.Toolchain
	for d := range r.catalog.Languages() {
		if !slices.Contains(a.Languages, d.Name) {
			continue
		}
		producer, ok := language.Role[language.Producer](r.catalog, d.Name)
		if !ok {
			return nil, lock.Lock{}, fmt.Errorf("%w: %s", ErrUnsupported, d.Name)
		}
		if !slices.Contains(toolchains, d.Toolchain) {
			toolchains = append(toolchains, d.Toolchain)
			if shared, ok := language.ToolchainRole[language.Producer](r.catalog, d.Toolchain); ok {
				units = append(units, render.Unit{Name: string(d.Toolchain), Producer: shared})
			}
		}
		languages = append(languages, d.Name)
		units = append(units, render.Unit{Name: string(d.Name), Producer: producer})
	}
	next.Answers.Languages = languages
	return units, next, nil
}

// resolve returns the content of .ergon.yaml, and the options of the configurable producers of
// units resolved from it and from previous, the lock of the repository or nil, for a. It returns
// the error of a file that does not read, and the errors of [options.Resolve].
func (r *Repository) resolve(units []render.Unit, a *language.Answers, previous *lock.Lock) (
	[]byte, options.Resolution, error,
) {
	file, _, err := r.read(language.Config)
	if err != nil {
		return nil, options.Resolution{}, err
	}
	var configurables []options.Producer
	for _, u := range units {
		if c, ok := u.Producer.(language.Configurable); ok {
			configurables = append(configurables, options.Producer{Name: u.Name, Configurable: c})
		}
	}
	var recorded map[string]any
	var answered *language.Answers
	if previous != nil {
		recorded, answered = previous.Settings, &previous.Answers
	}
	res, err := options.Resolve(file, recorded, answered, a, configurables, r.sections())
	return file, res, err
}

// resolved is a rendering and the record of the options that it rendered with.
type resolved struct {
	// record is the baseline value of each option, by its key in .ergon.yaml.
	record map[string]any

	rendering
}

// render resolves the options of units from .ergon.yaml and previous, the lock of the repository or
// nil, writes their sections into the content of .ergon.yaml, collects their contributions, renders
// their files for a, and merges the local files into them. It returns the errors that
// [Repository.targets] states for the options, the rendering and the local files.
func (r *Repository) render(units []render.Unit, a *language.Answers, previous *lock.Lock) (resolved, error) {
	file, res, err := r.resolve(units, a, previous)
	if err != nil {
		return resolved{}, err
	}
	var rend rendering
	if len(res.Sections) > 0 || len(res.Drop) > 0 {
		if rend.config, rend.configured, err = options.Write(file, res.Sections, res.Drop); err != nil {
			return resolved{}, err
		}
	}
	for i := range units {
		if j := slices.IndexFunc(
			res.Sections,
			func(s options.Section) bool { return s.Name == units[i].Name },
		); j >= 0 {
			units[i].Options = res.Sections[j].Options
		}
	}
	contributions, err := render.Collect(units)
	if err != nil {
		return resolved{}, err
	}
	files, err := render.Render(units, a, &contributions)
	if err != nil {
		return resolved{}, err
	}
	rend.files = make([]target, 0, len(files))
	for _, f := range files {
		if f.Path == language.Config {
			return resolved{}, fmt.Errorf("%w: %s renders it", ErrInvalidFile, f.Producer)
		}
		rend.files = append(rend.files, target{File: f})
	}
	if err := r.applyLocal(rend.files, units); err != nil {
		return resolved{}, err
	}
	return resolved{rendering: rend, record: res.Record}, nil
}

// sections returns the names of every section that .ergon.yaml can have: the names of the base
// producers, and of every language and toolchain of the catalog.
func (r *Repository) sections() []string {
	names := make([]string, 0, len(r.base))
	for _, p := range r.base {
		names = append(names, p.Name)
	}
	for d := range r.catalog.Languages() {
		for _, name := range []string{string(d.Name), string(d.Toolchain)} {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
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

// apply writes and removes the files of p, then writes next with p's entries of files when it
// differs from the lock in the repository. It returns the files that it wrote and removed, those
// before a failure included, and the error of the first write or removal that fails.
func (r *Repository) apply(p *plan, next *lock.Lock) ([]Change, error) {
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
	next.Files = p.files
	if next.Files == nil {
		next.Files = []lock.File{}
	}
	encoded := next.Encode()
	previous, _, err := r.read(lock.Path)
	if err != nil || bytes.Equal(previous, encoded) {
		return changes, err
	}
	if err := r.write(lock.Path, encoded); err != nil {
		return changes, err
	}
	return append(changes, Change{Path: lock.Path, Action: Wrote}), nil
}

// readLock returns the lock of the repository. It returns an error that wraps
// [ErrNotInitialized] for a repository without one, [lock.ErrInvalid] for a lock that does not
// parse, and [ErrNewerLock] for a lock that a newer release of ergon wrote than the running one.
//
// writableLock returns the lock of the repository, as [Repository.readLock] does, for a command
// that rewrites it. It returns the errors of readLock, and an error that wraps [ErrDevelopmentBuild]
// for a lock of a release in a build of ergon without a release.
func (r *Repository) writableLock() (lock.Lock, error) {
	l, err := r.readLock()
	if err != nil {
		return lock.Lock{}, err
	}
	if err := r.writable(&l); err != nil {
		return lock.Lock{}, err
	}
	return l, nil
}

// writable returns nil when the running ergon may write a lock over prev, the lock of the
// repository, which is nil for a repository without one. A release writes every lock. A build
// without a release, whose version no release has, writes only over a lock that such a build wrote,
// as in a repository that builds ergon from its own source. It returns an error that wraps
// [ErrDevelopmentBuild] for a new lock and for a lock of a release.
func (r *Repository) writable(prev *lock.Lock) error {
	if _, err := version.Parse(r.version); err == nil {
		return nil
	}
	if prev != nil && prev.Ergon == r.version {
		return nil
	}
	return fmt.Errorf("%w: this is ergon %s, so run a release of ergon", ErrDevelopmentBuild, r.version)
}

// A build of ergon without a release, whose version is dev, orders against no release. The running
// build reads every lock, and a lock that such a build wrote parses as the version 0.0.0, below every
// release of ergon.
func (r *Repository) readLock() (lock.Lock, error) {
	data, ok, err := r.read(lock.Path)
	if err != nil {
		return lock.Lock{}, err
	}
	if !ok {
		return lock.Lock{}, ErrNotInitialized
	}
	l, err := lock.Decode(data)
	if err != nil {
		return lock.Lock{}, err
	}
	running, err := version.Parse(r.version)
	wrote, _ := version.Parse(l.Ergon)
	if err == nil && wrote.Compare(running) > 0 {
		return lock.Lock{}, fmt.Errorf(
			"%w: ergon %s wrote the lock, and this is ergon %s, so install ergon %s or later",
			ErrNewerLock,
			l.Ergon,
			r.version,
			l.Ergon,
		)
	}
	return l, nil
}
