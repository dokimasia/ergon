// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"regexp"
)

// version matches the version of a release, and of a package of every kind but a Go module: a
// letter or a digit, then letters, digits, '.', '_', '+' and '-'. Each character is plain text in
// every file that ergon init renders, and in an argument of a command.
var version = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// Version is the version of a release that a section names outside its tools, such as the release
// of GNU make that a runner installs, or the release of the hooks of pre-commit, such as 4.4.1 or
// v6.0.0.
type Version string

// Validate returns an error that wraps [ErrInvalid] for a v that is empty, that has a character
// other than a letter, a digit, '.', '_', '+' and '-', or that starts with '.', '_', '+' or '-'.
func (v Version) Validate() error {
	if !version.MatchString(string(v)) {
		return fmt.Errorf("%w: version %q, which is not the version of a release", ErrInvalid, v)
	}
	return nil
}
