// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrInvalidAction is the error for an action that is not pinned to the commit of a release.
var ErrInvalidAction = errors.New("workflow: invalid action")

// The rules of the fields of an [Action].
var (
	// uses matches owner/name, and owner/name/path for an action in a directory of its repository.
	uses = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)

	// commit matches the full SHA-1 of a commit, in lowercase.
	commit = regexp.MustCompile(`^[0-9a-f]{40}$`)

	// release matches the name of a release, such as v7.0.1 or 2.37.2.
	release = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)
)

// Action is an action of a workflow, pinned to the commit of a release. A step refers to it as
// uses@commit, with the release in a comment, so a tag that moves cannot change what the step
// runs. In .ergon.yaml an action is a mapping of the keys uses, commit and release.
type Action struct {
	// Uses is the repository of the action as owner/name, such as actions/checkout, or
	// owner/name/path for an action in a directory of its repository.
	Uses string `yaml:"uses"`

	// Commit is the full SHA-1 of the commit that the release tags, as 40 lowercase hexadecimal
	// digits.
	Commit string `yaml:"commit"`

	// Release is the name of the release, such as v7.0.1.
	Release string `yaml:"release"`
}

// Path returns the action in the directory path of the repository of a, at the same commit and
// release: github/codeql-action with the path init returns github/codeql-action/init.
func (a Action) Path(path string) Action {
	a.Uses += "/" + path
	return a
}

// String returns the reference of a in a step: uses@commit, a space, and the release in a
// comment, such as "actions/checkout@3d3c…90b1 # v7.0.1".
func (a Action) String() string {
	return a.Uses + "@" + a.Commit + " # " + a.Release
}

// Validate returns an error that wraps [ErrInvalidAction] for a Uses that is not owner/name or
// owner/name/path, a Commit that is not 40 lowercase hexadecimal digits, and a Release that is
// empty or has a character other than a letter, a digit, '.', '_', '+' or '-'. It returns nil for
// a valid action.
func (a Action) Validate() error {
	if !uses.MatchString(a.Uses) {
		return fmt.Errorf("%w: uses %q, which is not owner/name or owner/name/path", ErrInvalidAction, a.Uses)
	}
	if !commit.MatchString(a.Commit) {
		return fmt.Errorf("%w: commit %q of %s, which is not 40 lowercase hexadecimal digits", ErrInvalidAction,
			a.Commit, a.Uses)
	}
	if !release.MatchString(a.Release) {
		return fmt.Errorf("%w: release %q of %s, which is not the name of a release", ErrInvalidAction, a.Release,
			a.Uses)
	}
	return nil
}
