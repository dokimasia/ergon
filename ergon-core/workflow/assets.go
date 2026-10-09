// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrInvalidAssets is the error for release assets that release.yml cannot publish.
var ErrInvalidAssets = errors.New("workflow: invalid assets")

// tap matches a repository on GitHub as owner/name: an owner of letters, digits and '-' that starts
// with a letter or a digit, a slash, and a name of letters, digits, '.', '_' and '-'.
var tap = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// Assets states that a producer builds the assets of releases in the job pack of release.yml, such
// as the archives of the commands of a Go module. The job pack then attests the assets, and the
// publish attaches them to the release of their tag.
type Assets struct {
	// Tap is the Homebrew tap on GitHub, as owner/name, to which the job homebrew of release.yml
	// commits the casks of the assets, or empty for assets without a cask.
	Tap string
}

// Validate returns an error that wraps [ErrInvalidAssets] for a Tap that is neither empty nor
// owner/name. It returns nil for valid assets.
func (a *Assets) Validate() error {
	if a.Tap != "" && !tap.MatchString(a.Tap) {
		return fmt.Errorf("%w: tap %q, which is not owner/name", ErrInvalidAssets, a.Tap)
	}
	return nil
}
