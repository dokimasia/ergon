// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language

import (
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/workspace"
)

// The shared files of ergon init: the files that the common files or the GitHub files open with a
// fragment, and to which every language may contribute its own fragment.
const (
	// EditorConfig is the editor configuration, with a section for the file types of a language.
	EditorConfig = ".editorconfig"

	// GitAttributes are the attributes of git, with the diff driver of a language's files.
	GitAttributes = ".gitattributes"

	// GitIgnore is the list of files that git ignores, with the build output of a language.
	GitIgnore = ".gitignore"

	// Makefile is the gate of the repository, with the fmt, lint, test, audit and check targets of
	// a language.
	Makefile = "Makefile"

	// CI is the workflow of the gate, with the job check-<language> of a language, which runs
	// make check-<language>. A language's jobs follow the jobs of the GitHub files under the key
	// jobs, indented by two spaces.
	CI = ".github/workflows/ci.yml"

	// Security is the workflow of the security checks, with the CodeQL job of a language that
	// CodeQL analyzes. The job calls the reusable workflow .github/workflows/codeql.yml.
	Security = ".github/workflows/security.yml"

	// Dependabot is the configuration of Dependabot, with an entry under the key updates for the
	// package manager of a language.
	Dependabot = ".github/dependabot.yml"
)

// Checkout is the action that checks out the repository in every job of the workflows of ergon
// init: actions/checkout at the commit of a release, with the release in a comment. Every job runs
// it with persist-credentials set to false, so the git configuration of the job has no token.
const Checkout = "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1"

// Go is the release of Go that a job installs to run a tool with go run in a repository whose
// own toolchain is not Go, such as commitlint in the check of the commit messages and osv-scanner
// in the gate of Java and Kotlin.
const Go = "1.27.1"

// The runner images of the workflows of ergon init, each pinned to a version of its system, so an
// image changes only with a release of ergon.
const (
	// Linux is the runner of the jobs that check text, such as the Markdown or the commit messages,
	// whose result does not depend on the system.
	Linux = "ubuntu-26.04"

	// MacOS is the runner of macOS on arm64.
	MacOS = "macos-26"

	// Windows is the runner of Windows on x64.
	Windows = "windows-2025"

	// Runners is the matrix of the jobs that build, test or run ergon: Linux, macOS and Windows.
	Runners = "[" + Linux + ", " + MacOS + ", " + Windows + "]"
)

// The placeholders of the template of a job, which [Job] fills.
const (
	checkout  = "{{checkout}}"
	runners   = "{{runners}}"
	goVersion = "{{go-version}}"
)

// jobs fills the placeholders of the template of a job.
var jobs = strings.NewReplacer(checkout, Checkout, runners, Runners, goVersion, Go)

// Job returns the template of a job of a workflow with its placeholders filled: each {{checkout}}
// with [Checkout], each {{runners}} with [Runners], and each {{go-version}} with [Go].
func Job(template string) string {
	return jobs.Replace(template)
}

// Class is how ergon init treats a file that it renders. The zero value is not a valid class.
type Class uint8

const (
	// Managed is a file that init renders whole and checks against its lock. A repository changes
	// it only through the answers and its local file.
	Managed Class = 1

	// Configured is a file whose keys init writes. Every other key of the file belongs to the
	// repository, as in .ergon.yaml.
	Configured Class = 2

	// Seeded is a file that init writes when it is absent, and that the repository maintains from
	// then on, such as README.md.
	Seeded Class = 3
)

// Valid reports whether c is Managed, Configured or Seeded.
func (c Class) Valid() bool {
	return c == Managed || c == Configured || c == Seeded
}

// Repository is a repository on GitHub, as owner/name, such as "dokimasia/ergon".
type Repository string

// repository matches a repository on GitHub as owner/name.
var repository = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// Valid reports whether r is owner/name: an owner of letters, digits and hyphens that starts with a
// letter or a digit, a slash, and a name of letters, digits, dots, hyphens and underscores.
func (r Repository) Valid() bool {
	return repository.MatchString(string(r))
}

// Answers are the answers of ergon init. The lock of a repository records them, and every
// [Initializer] renders its files from them. The fields are in the order that packs their
// pointers first, which is also the order of their keys in the lock.
type Answers struct {
	// Name is the name of the repository, such as "ergon".
	Name string `json:"name"`

	// Owner is the copyright holder, such as "Dokimasia B.V.".
	Owner string `json:"owner"`

	// License is the SPDX identifier of the license, such as "MIT".
	License string `json:"license"`

	// Repository is the repository on GitHub.
	Repository Repository `json:"repository"`

	// SecurityContact is the address that receives reports of vulnerabilities.
	SecurityContact string `json:"security-contact"`

	// Languages are the languages of the repository, in the order of the catalog.
	Languages []workspace.Language `json:"languages"`

	// Year is the year of the copyright notice: the year of the repository's first release.
	Year int `json:"year"`
}

// File is a file, or a fragment of a shared file, that an [Initializer] renders.
type File struct {
	// Path is the path of the file in the repository: relative, slash-separated and clean.
	Path string

	// Content is the whole file, which no other producer renders. It is nil when Fragment is set.
	Content []byte

	// Fragment is this producer's part of a file that several producers render. init joins the
	// fragments in the order of the producers. It is nil when Content is set.
	Fragment []byte

	// Class is how init treats the file.
	Class Class
}

// Initializer renders the files that a producer contributes to a repository: the common files,
// the GitHub files, or the files of a language. A language registers it as a role.
type Initializer interface {
	// Files returns the files and the fragments that the producer contributes for a, which it
	// reads and does not modify. It reads no file and runs no command, so a rendering depends only
	// on a and on the release of ergon. It returns an error for answers that it cannot render, such
	// as an unsupported license.
	Files(a *Answers) ([]File, error)
}

// Fixed is an [Initializer] whose files do not depend on the answers, such as the fragments that a
// language contributes to the shared files.
type Fixed []File

var _ Initializer = Fixed(nil)

// Files returns a copy of the list of files, whatever the answers, which may be nil. The copy
// shares the bytes of each file, which no caller changes. It never returns an error.
func (f Fixed) Files(*Answers) ([]File, error) {
	return slices.Clone(f), nil
}
