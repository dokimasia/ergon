// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Versioner is the [language.Versioner] of the toolchain terraform. A package of Terraform has no
// manifest with a version and no requirement on another package of the repository. The zero value
// is ready for use, and it is safe for concurrent use.
type Versioner struct{}

var _ language.Versioner = Versioner{}

// Resolve returns an error that wraps [ErrRequirement] for every req.
func (Versioner) Resolve(req string, _ version.Version) (language.Resolution, error) {
	return 0, fmt.Errorf("%w: %q", ErrRequirement, req)
}

// Rewrite returns an error that wraps [ErrRequirement] for every req.
func (Versioner) Rewrite(req string, _ version.Version) (string, error) {
	return "", fmt.Errorf("%w: %q", ErrRequirement, req)
}

// Validate returns nil for every release, because Terraform and the Terraform Registry read every
// version of Semantic Versioning 2.0.0 from a tag.
func (Versioner) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply does not change a file and returns an empty list of paths, because a package of Terraform
// has no manifest. The command writes the CHANGELOG.md of the package, whose headings record its
// versions.
func (Versioner) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return []string{}, nil
}
