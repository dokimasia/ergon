// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// cacheID matches the ID of a cache: a lowercase letter, then lowercase letters, digits and '-'.
var cacheID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Cache is a directory that a job keeps in the cache of GitHub Actions between its runs. The
// producer of the GitHub files restores the directory before the steps of the job and saves it
// after them, with the action cache of the section github.
//
// The key of a cache starts with its ID, the system, the architecture, the job, the runtime version
// of a matrix and, for a cache with Days, the number of whole intervals since the Unix epoch. That
// prefix is the restore key, so a job restores the newest cache of the prefix when its key misses:
//
//   - A cache with Files ends its key in the digest of the files. The job saves the cache after its
//     steps when the key missed, as actions/cache does.
//   - A cache without Files ends its key in the ID and the attempt of the run. The job saves the
//     cache after its steps in every run, also after a step that fails.
type Cache struct {
	// ID starts the key of the cache and the IDs of its steps, such as golangci-lint.
	ID string

	// Name is what the directory contains, which the names of the steps of the cache state, such as
	// the cache of golangci-lint.
	Name string

	// Path is the directory, as the input path of actions/cache takes it, such as an expression of
	// GitHub Actions that selects the directory of each system.
	Path string

	// Input names a boolean input of workflow_dispatch, which is true by default, or is empty. A run
	// of workflow_dispatch with the input false skips the restore, so its steps start with an empty
	// directory. Only a job of nightly.yml has a cache with an input.
	Input string

	// Files are the patterns of hashFiles whose digest ends the key, such as **/go.sum, or none for
	// a cache that the job saves in every run.
	Files []string

	// Days is the length of an interval of the key in days, or 0 for a key without an interval. A
	// new interval starts an empty cache, so the directory does not grow without a bound.
	Days int
}

// Validate returns an error that wraps [ErrInvalidJob] for the first value of c that a workflow
// cannot contain:
//
//   - an ID that is not a lowercase letter followed by lowercase letters, digits and '-'
//   - an empty Name or Path, or a Name or a Path that spans lines
//   - a pattern of Files that is empty, spans lines or has a single quote
//   - a Days below 0
//   - an Input that is not empty and not a letter or '_' followed by letters, digits, '_' and '-'
func (c *Cache) Validate() error {
	if !cacheID.MatchString(c.ID) {
		return fmt.Errorf("%w: cache id %q, which is not a lowercase letter followed by lowercase letters, digits "+
			"and '-'", ErrInvalidJob, c.ID)
	}
	for _, v := range []string{c.Name, c.Path} {
		if v == "" || strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("%w: the cache %s has an empty name or path, or one that spans lines", ErrInvalidJob,
				c.ID)
		}
	}
	for _, f := range c.Files {
		if f == "" || strings.ContainsAny(f, "'\r\n") {
			return fmt.Errorf("%w: the cache %s has the files %q, which are empty, span lines or have a single quote",
				ErrInvalidJob, c.ID, f)
		}
	}
	if c.Days < 0 {
		return fmt.Errorf("%w: the cache %s has an interval of %d days, which is less than 0", ErrInvalidJob, c.ID,
			c.Days)
	}
	if c.Input != "" && !identifier.MatchString(c.Input) {
		return fmt.Errorf("%w: the cache %s has the input %q, which is not an identifier", ErrInvalidJob, c.ID,
			c.Input)
	}
	return nil
}
