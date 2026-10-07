// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// ErrInvalidCodeQL is the error for a CodeQL analysis that a workflow cannot contain.
var ErrInvalidCodeQL = errors.New("workflow: invalid CodeQL analysis")

// language matches a name of a language of CodeQL that is also part of a job's identifier, such as
// go or javascript-typescript.
var language = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// buildModes are the ways in which codeql.yml lets CodeQL extract a language. The workflow runs no
// build of its own, so it has no build mode manual.
var buildModes = []string{"none", "autobuild"}

// CodeQL is the CodeQL analysis of one language, which security.yml runs as the job
// codeql-<Language> through the reusable workflow codeql.yml.
type CodeQL struct {
	// Language is the name of the language in CodeQL, such as go, java-kotlin or
	// javascript-typescript.
	Language string

	// Name is the name of the language in the checks of a pull request, such as Java and Kotlin.
	Name string

	// BuildMode is how CodeQL extracts the language: none, which reads the sources alone, or
	// autobuild, which builds them with the commands that CodeQL derives.
	BuildMode string

	// Files is a pattern of hashFiles, such as go.work. The analysis runs only when the repository
	// has a file that matches it, which is the file that pins the language's toolchain.
	Files string

	// Timeout is the limit of the analysis in minutes.
	Timeout int
}

// Validate returns an error that wraps [ErrInvalidCodeQL] for the first value of c that a workflow
// cannot contain: a Language that is not a lowercase letter followed by lowercase letters, digits
// and '-', an empty Name or one that spans lines, a BuildMode other than none and autobuild, an
// empty Files or one that spans lines or has a single quote, and a Timeout below 1. CodeQL itself
// rejects a language that it does not analyze, when the analysis runs.
func (c *CodeQL) Validate() error {
	if !language.MatchString(c.Language) {
		return fmt.Errorf("%w: language %q, which is not a name of a language", ErrInvalidCodeQL, c.Language)
	}
	if c.Name == "" || strings.ContainsAny(c.Name, "\r\n") {
		return fmt.Errorf("%w: %s has an empty name or one that spans lines", ErrInvalidCodeQL, c.Language)
	}
	if !slices.Contains(buildModes, c.BuildMode) {
		return fmt.Errorf("%w: %s has the build mode %q, which is none of %s", ErrInvalidCodeQL, c.Language,
			c.BuildMode, strings.Join(buildModes, ", "))
	}
	if c.Files == "" || strings.ContainsAny(c.Files, "'\r\n") {
		return fmt.Errorf("%w: %s has the files %q, which are empty, span lines or have a single quote",
			ErrInvalidCodeQL, c.Language, c.Files)
	}
	if c.Timeout < 1 {
		return fmt.Errorf("%w: %s has the timeout %d, which is less than a minute", ErrInvalidCodeQL, c.Language,
			c.Timeout)
	}
	return nil
}
