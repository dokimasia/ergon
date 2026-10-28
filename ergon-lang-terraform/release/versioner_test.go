// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/terraform/release"
)

// pkg is the package of a repository of Terraform at version 1.2.0.
var pkg = workspace.Package{Name: "terraform-aws-vpc", Toolchain: "terraform", Dir: ".", Version: version.Version{
	Major: 1, Minor: 2,
}}

func TestVersioner(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrRequirement for a requirement", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Resolve("v1.2.0", pkg.Version)
			assert.ErrorIs(t, err, release.ErrRequirement, "Resolve")
		})
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrRequirement for a requirement", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Rewrite("v1.2.0", pkg.Version)
			assert.ErrorIs(t, err, release.ErrRequirement, "Rewrite")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a major release", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, release.Versioner{}.Validate(&pkg, version.Version{Major: 2}), "Validate")
		})

		t.Run("returns nil for a pre-release", func(t *testing.T) {
			t.Parallel()
			v := version.Version{Major: 1, Minor: 3, Pre: "rc.1"}
			assert.NoError(t, release.Versioner{}.Validate(&pkg, v), "Validate")
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("changes no file of the repository", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			edits := []language.Edit{{Version: version.Version{Major: 1, Minor: 3}, Package: pkg}}
			paths, err := release.Versioner{}.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			assert.Empty(t, paths, "the changed paths")
		})
	})
}
