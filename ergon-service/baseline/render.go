// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
)

// ErrInvalidFile is the error for a file that a producer renders wrong: a path that is not
// relative, clean and slash-separated, a path under .ergon, an invalid class, both or neither of
// Content and Fragment, or a path that two producers render as different files. It is a defect of
// the producer.
var ErrInvalidFile = errors.New("baseline: invalid file")

// reserved is the directory of the lock and the local files, which no producer renders into.
const reserved = ".ergon/"

// Producer is a named source of the files of a repository: the common files, the GitHub files or
// a language. The lock records the name of the producer of each managed file.
type Producer struct {
	// Initializer renders the files of the producer.
	Initializer language.Initializer

	// Name identifies the producer in the lock, such as "common" or "go".
	Name string
}

// target is one file of a rendering: the whole file, or the joined fragments of a shared file.
type target struct {
	// path is the path of the file in the repository.
	path string

	// producer is the producer of the file, or of its first fragment.
	producer string

	// local is the digest of the local file that content contains, or empty.
	local string

	// content is the rendering, with the local file merged into a managed file.
	content []byte

	// class is how a command treats the file.
	class language.Class

	// shared reports that the file is joined from fragments.
	shared bool
}

// render returns the files that the producers render for a, sorted by path. It joins the
// fragments of a shared file in the order of the producers. It returns the error of a producer,
// and an error that wraps [ErrInvalidFile] for a file that a producer renders wrong.
func render(producers []Producer, a *language.Answers) ([]target, error) {
	var targets []target
	for _, p := range producers {
		files, err := p.Initializer.Files(a)
		if err != nil {
			return nil, fmt.Errorf("baseline: the files of %s do not render: %w", p.Name, err)
		}
		for _, f := range files {
			if err := validate(f); err != nil {
				return nil, fmt.Errorf("%w: %s renders %q: %w", ErrInvalidFile, p.Name, f.Path, err)
			}
			i := slices.IndexFunc(targets, func(t target) bool { return t.path == f.Path })
			if i < 0 {
				targets = append(targets, target{
					path:     f.Path,
					class:    f.Class,
					content:  slices.Concat(f.Content, f.Fragment),
					producer: p.Name,
					shared:   f.Fragment != nil,
				})
				continue
			}
			t := &targets[i]
			if !t.shared || f.Fragment == nil || t.class != f.Class {
				return nil, fmt.Errorf("%w: %s and %s both render %q", ErrInvalidFile, t.producer, p.Name, f.Path)
			}
			t.content = append(t.content, f.Fragment...)
		}
	}
	slices.SortFunc(targets, func(a, b target) int { return strings.Compare(a.path, b.path) })
	return targets, nil
}

// errContent, errPath and errClass are the causes that [ErrInvalidFile] wraps.
var (
	errContent = errors.New("exactly one of Content and Fragment must be set")
	errPath    = errors.New("the path must be relative, clean, slash-separated and outside .ergon")
	errClass   = errors.New("the class is not Managed, Configured or Seeded")
)

// validate returns the rule of a rendered file that f breaks, or nil.
func validate(f language.File) error {
	if (f.Content == nil) == (f.Fragment == nil) {
		return errContent
	}
	if !fs.ValidPath(f.Path) || f.Path == "." || strings.HasPrefix(f.Path+"/", reserved) {
		return errPath
	}
	if !f.Class.Valid() {
		return errClass
	}
	return nil
}
