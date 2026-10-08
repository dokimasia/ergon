// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/core/workspace"
)

// Config is .ergon.yaml, the configuration of every ergon command. ergon init writes a section of
// it for each producer that has options.
const Config = ".ergon.yaml"

// ErrInvalidAnswer is the error of [Answers.Validate] for an answer that no producer can render.
var ErrInvalidAnswer = errors.New("language: invalid answer")

// ErrInvalidLocal is the error of a [LocalChecker] for a local file that the repository may not
// have.
var ErrInvalidLocal = errors.New("language: invalid local file")

// repository matches a repository on GitHub as owner/name.
var repository = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// Repository is a repository on GitHub, as owner/name, such as "dokimasia/ergon".
type Repository string

// Valid reports whether r is owner/name: an owner of letters, digits and hyphens that starts with a
// letter or a digit, a slash, and a name of letters, digits, dots, hyphens and underscores.
func (r Repository) Valid() bool {
	return repository.MatchString(string(r))
}

// Answers are the answers of ergon init. The lock of a repository records them, and every
// [Producer] renders its files from them. The fields are in the order of their keys in the lock.
type Answers struct {
	// Name is the name of the repository, such as "ergon".
	Name string `json:"name"`

	// Owner is the copyright holder, such as "Dokimasia B.V.".
	Owner string `json:"owner"`

	// License is the SPDX identifier of the license, such as "MIT".
	License spdx.ID `json:"license"`

	// Repository is the repository on GitHub.
	Repository Repository `json:"repository"`

	// SecurityContact is the address that receives reports of vulnerabilities.
	SecurityContact string `json:"security-contact"`

	// Languages are the languages of the repository, in the order of the catalog.
	Languages []workspace.Language `json:"languages"`

	// Year is the year of the copyright notice: the year of the repository's first release.
	Year int `json:"year"`
}

// Validate returns an error that wraps [ErrInvalidAnswer] for the first answer of a that no
// producer can render: a Name, an Owner or a SecurityContact that is blank or spans lines, a
// License that is not an identifier of [spdx.IDs], a Year outside 1 to 9999, and a Repository that
// is not owner/name, as [Repository.Valid] states. The command checks the languages against its
// catalog. Validate reads a and does not modify it.
func (a *Answers) Validate() error {
	lines := []struct{ answer, value string }{
		{answer: "name", value: a.Name},
		{answer: "owner", value: a.Owner},
		{answer: "security contact", value: a.SecurityContact},
	}
	for _, l := range lines {
		if strings.TrimSpace(l.value) == "" || strings.ContainsAny(l.value, "\r\n") {
			return fmt.Errorf("%w: the %s %q, which is not one line of text", ErrInvalidAnswer, l.answer, l.value)
		}
	}
	if !a.License.Valid() {
		return fmt.Errorf("%w: license %q, which is not the SPDX identifier of a license of ergon", ErrInvalidAnswer,
			a.License)
	}
	if a.Year < 1 || a.Year > 9999 {
		return fmt.Errorf("%w: year %d, which is not between 1 and 9999", ErrInvalidAnswer, a.Year)
	}
	if !a.Repository.Valid() {
		return fmt.Errorf("%w: repository %q, which is not owner/name", ErrInvalidAnswer, a.Repository)
	}
	return nil
}

// Producer renders the files of one concern of a repository: the common files, the GitHub files,
// the license files, a toolchain that two languages share, or a language. A toolchain and a
// language register it as a role. Its templates read the answers, its options and the
// contributions of every producer, and a [Calculator] adds the values that it computes.
type Producer interface {
	// Templates returns the templates of the producer. Their tree mirrors the repository under
	// managed/, seeded/ and shared/: a managed file, a seeded file, and the producer's fragment of a
	// file that several producers render. Each name ends in .tmpl.
	Templates() fs.FS
}

// Calculator is a producer whose templates read values that it computes from the answers, the
// options and the contributions.
type Calculator interface {
	// Data returns the values that the templates of the producer read as .Data. a are the answers,
	// o are the producer's options, nil for a producer that is not [Configurable], and c are the
	// contributions of every producer to the workflows. Data reads no file and runs no command, so a
	// rendering depends on a, o, c and the release of ergon alone, and it modifies none of them. It
	// returns an error for answers, options or contributions that the producer cannot render.
	Data(a *Answers, o Options, c *workflow.Contribution) (any, error)
}

// Options are the options of a producer's section of .ergon.yaml: a pointer to a struct whose
// fields have yaml tags and doc tags, and types of go.dokimi.dev/ergon/core/option wherever one
// states the option.
type Options interface {
	// Validate returns an error for the first value that the producer cannot render, by a rule of
	// the producer, such as a step of check that the producer does not have. The command checks the
	// value of each field whose type has a Validate method before it calls Validate.
	Validate() error
}

// Configurable is a producer with a section of .ergon.yaml.
type Configurable interface {
	// Options returns a new value of the producer's options at the baseline.
	Options() Options
}

// Contributor is a producer with a part of the workflows: jobs of ci.yml, a CodeQL analysis, or a
// package manager that Dependabot updates.
type Contributor interface {
	// Contribution returns the producer's part of the workflows for o, which is nil for a producer
	// that is not [Configurable].
	Contribution(o Options) workflow.Contribution
}

// Placer is a producer that renders managed files at paths that its options state, beside the
// files of its templates, such as the LICENSE of each directory that the section license lists. A
// template tree cannot mirror a path that an option states.
type Placer interface {
	// Files returns the files of the producer for the answers a, its options o, nil for a producer
	// that is not [Configurable], and the contributions c of every producer to the workflows. Files
	// reads no file and runs no command, as [Calculator.Data] does, and it modifies none of a, o and
	// c. It returns an error for answers or options that the producer cannot render.
	Files(a *Answers, o Options, c *workflow.Contribution) ([]File, error)
}

// File is a managed file that a [Placer] renders.
type File struct {
	// Path is the path of the file in the repository: relative, clean and slash-separated, such as
	// enterprise/LICENSE.
	Path string

	// Content is the content of the file.
	Content []byte
}

// LocalChecker is a producer that checks the local files of the managed files that it renders,
// such as a local file that would let a tool edit another managed file.
type LocalChecker interface {
	// CheckLocal returns an error that wraps [ErrInvalidLocal] for content, the managed file at path
	// with its local file merged in, that the producer refuses. path is relative, clean and
	// slash-separated. CheckLocal reads no file and runs no command, and it modifies no content.
	CheckLocal(path string, content []byte) error
}
