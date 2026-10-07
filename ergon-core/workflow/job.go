// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// ErrInvalidJob is the error for a job, or the setup of a job, that a workflow cannot contain.
var ErrInvalidJob = errors.New("workflow: invalid job")

// The rules of the names of a job.
var (
	// runner matches the label of a runner image, such as ubuntu-26.04.
	runner = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

	// toolchain matches the name of a toolchain: a lowercase letter, then lowercase letters and
	// digits.
	toolchain = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

	// scope matches the name of a scope of the permissions of a job, such as security-events.
	scope = regexp.MustCompile(`^[a-z][a-z-]*$`)
)

// accesses are the levels of access that a job grants to a scope of its permissions.
var accesses = []string{"read", "write", "none"}

// Setup is how a job of ci.yml sets up a toolchain. A language whose toolchain is its own states
// it in [Job.Setup]. A toolchain that two languages share states it in [Contribution.Setup], and
// the jobs of both languages run it.
type Setup struct {
	// Files is a pattern of hashFiles, such as go.work or **/go.mod. The steps that follow the
	// checkout run only when the repository has a file that matches it, so a repository without
	// the toolchain's files passes the job. It is empty for a job whose steps always run.
	Files string

	// Runners are the runner images of the job's matrix. An empty list selects every runner of the
	// section github of .ergon.yaml.
	Runners []string

	// Versions are the runtime versions of the job's matrix, which the setup steps read as
	// matrix.version. An empty list runs the version that the toolchain's pin file states.
	Versions []string

	// Env are the environment variables of the job, by name.
	Env map[string]string

	// Steps are the steps that install the toolchain, after the checkout, the installation of GNU
	// make and the installation of ergon.
	Steps []Step

	// Timeout is the limit of the job in minutes.
	Timeout int
}

// Validate returns an error that wraps [ErrInvalidJob] for the first value of s that a workflow
// cannot contain: a Files that spans lines or has a single quote, a runner that is not a label of
// letters, digits, '.', '_' and '-', a version that is empty or spans lines, a runner or a version
// that the list names twice, a Timeout below 1, an environment variable whose name is not a letter
// or '_' followed by letters, digits and '_', and a step that is not valid, as [Step.Validate]
// states.
func (s *Setup) Validate() error {
	if strings.ContainsAny(s.Files, "'\r\n") {
		return fmt.Errorf("%w: files %q, which spans lines or has a single quote", ErrInvalidJob, s.Files)
	}
	for i, r := range s.Runners {
		if !runner.MatchString(r) || slices.Contains(s.Runners[:i], r) {
			return fmt.Errorf("%w: runner %q, which is not a label or is named twice", ErrInvalidJob, r)
		}
	}
	for i, v := range s.Versions {
		if v == "" || strings.ContainsAny(v, "\r\n") || slices.Contains(s.Versions[:i], v) {
			return fmt.Errorf("%w: version %q, which is empty, spans lines or is named twice", ErrInvalidJob, v)
		}
	}
	if s.Timeout < 1 {
		return fmt.Errorf("%w: timeout %d, which is less than a minute", ErrInvalidJob, s.Timeout)
	}
	for _, name := range slices.Sorted(maps.Keys(s.Env)) {
		if !variable.MatchString(name) {
			return fmt.Errorf("%w: environment variable %q, which is not a name of a variable", ErrInvalidJob, name)
		}
	}
	for i := range s.Steps {
		if err := s.Steps[i].Validate(); err != nil {
			return fmt.Errorf("%w: setup step %d: %w", ErrInvalidJob, i+1, err)
		}
	}
	return nil
}

// Job is a job of ci.yml. The producer of the GitHub files renders every job from one skeleton:
// the checkout, then the installation of GNU make for a job with a setup, the installation of
// ergon for a job that runs it, the setup steps, and the steps of the job.
//
// A job runs in one of three ways:
//
//   - with a setup, its own in Setup or the setup of the toolchain that Toolchain names, on the
//     runners and the versions of the setup
//   - as a check of text, which Text states, on the Linux runner of the section github
//   - on every runner of the section github otherwise
type Job struct {
	// ID is the key of the job in the workflow, such as check-go.
	ID string

	// Name is the name of the job in the checks of a pull request, such as Go. The renderer adds
	// the runner and the version of a job that runs on a matrix.
	Name string

	// If is the condition of the job, or empty for a job that runs on every event of the workflow.
	If string

	// Toolchain names the producer of the toolchain whose [Contribution.Setup] the job runs, such
	// as jvm, or is empty.
	Toolchain string

	// Setup is the job's own setup of its toolchain, or nil.
	Setup *Setup

	// Permissions are the access of the job's GITHUB_TOKEN to each scope, such as contents: read.
	Permissions map[string]string

	// Steps are the steps of the job after its setup.
	Steps []Step

	// Timeout is the limit in minutes of a job without a setup, and 0 for a job with one, which the
	// timeout of its setup limits.
	Timeout int

	// Text reports that the job checks text, such as Markdown or commit messages, whose result does
	// not depend on the system.
	Text bool

	// History reports that the checkout fetches every commit, as the check of the commit messages of
	// a pull request needs.
	History bool

	// Ergon reports that the job runs ergon, whose release the lock names. A job with a setup runs
	// it through the targets of the Makefile, whatever Ergon states.
	Ergon bool
}

// Validate returns an error that wraps [ErrInvalidJob] for the first value of j that a workflow
// cannot contain:
//
//   - an ID that is not a letter or '_' followed by letters, digits, '_' and '-'
//   - an empty Name, or a Name or an If that spans lines
//   - a Toolchain that is not a lowercase letter followed by lowercase letters and digits
//   - more than one of Setup, Toolchain and Text
//   - a Timeout other than 0 for a job with a setup, and below 1 for any other job
//   - a setup that is not valid, as [Setup.Validate] states
//   - a scope of the permissions that is not lowercase letters and '-', or an access other than
//     read, write and none
//   - no step, or a step that is not valid, as [Step.Validate] states
//
// It reads the permissions in the order of their scopes, so it returns the same error for the same
// job.
func (j *Job) Validate() error {
	if !identifier.MatchString(j.ID) {
		return fmt.Errorf("%w: id %q, which is not an identifier", ErrInvalidJob, j.ID)
	}
	if j.Name == "" || strings.ContainsAny(j.Name, "\r\n") || strings.ContainsAny(j.If, "\r\n") {
		return fmt.Errorf("%w: %s has an empty name, or a name or a condition that spans lines", ErrInvalidJob, j.ID)
	}
	if j.Toolchain != "" && !toolchain.MatchString(j.Toolchain) {
		return fmt.Errorf("%w: %s names the toolchain %q, which is not a name", ErrInvalidJob, j.ID, j.Toolchain)
	}
	setup := j.Setup != nil || j.Toolchain != ""
	if (j.Setup != nil && j.Toolchain != "") || (setup && j.Text) {
		return fmt.Errorf("%w: %s has more than one of a setup, a toolchain and a check of text", ErrInvalidJob, j.ID)
	}
	if (setup && j.Timeout != 0) || (!setup && j.Timeout < 1) {
		return fmt.Errorf("%w: %s has the timeout %d, which a job with a setup leaves 0 and any other job sets "+
			"to at least a minute", ErrInvalidJob, j.ID, j.Timeout)
	}
	if j.Setup != nil {
		if err := j.Setup.Validate(); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidJob, j.ID, err)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(j.Permissions)) {
		if access := j.Permissions[name]; !scope.MatchString(name) || !slices.Contains(accesses, access) {
			return fmt.Errorf("%w: %s grants %s %q, which is not a scope with read, write or none", ErrInvalidJob, j.ID,
				name, access)
		}
	}
	if len(j.Steps) == 0 {
		return fmt.Errorf("%w: %s has no step", ErrInvalidJob, j.ID)
	}
	for i := range j.Steps {
		if err := j.Steps[i].Validate(); err != nil {
			return fmt.Errorf("%w: %s: step %d: %w", ErrInvalidJob, j.ID, i+1, err)
		}
	}
	return nil
}
