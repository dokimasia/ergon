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

// ErrInvalidUpdate is the error for an update of Dependabot that dependabot.yml cannot contain.
var ErrInvalidUpdate = errors.New("workflow: invalid update")

// ecosystem matches a package ecosystem of Dependabot, such as gomod or github-actions.
var ecosystem = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Update is an entry of dependabot.yml: the weekly updates of the manifests of one package manager.
type Update struct {
	// Ecosystem is the package ecosystem of Dependabot, such as gomod, gradle or npm.
	Ecosystem string

	// Directories are the directories of the manifests, from the root of the repository, such as /
	// or the glob /**/* for a manifest in every directory.
	Directories []string
}

// Validate returns an error that wraps [ErrInvalidUpdate] for the first value of u that
// dependabot.yml cannot contain: an Ecosystem that is not a lowercase letter followed by lowercase
// letters, digits and '-', no directory, and a directory that does not start with /, spans lines or
// is named twice. Dependabot itself rejects an ecosystem that it does not update.
func (u *Update) Validate() error {
	if !ecosystem.MatchString(u.Ecosystem) {
		return fmt.Errorf("%w: ecosystem %q, which is not a name of an ecosystem", ErrInvalidUpdate, u.Ecosystem)
	}
	if len(u.Directories) == 0 {
		return fmt.Errorf("%w: %s has no directory", ErrInvalidUpdate, u.Ecosystem)
	}
	for i, dir := range u.Directories {
		if !strings.HasPrefix(dir, "/") || strings.ContainsAny(dir, "\r\n") || slices.Contains(u.Directories[:i], dir) {
			return fmt.Errorf("%w: %s has the directory %q, which does not start with /, spans lines or is named "+
				"twice", ErrInvalidUpdate, u.Ecosystem, dir)
		}
	}
	return nil
}
