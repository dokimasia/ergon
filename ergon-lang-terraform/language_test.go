// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package terraform_test

import (
	"context"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
	"go.dokimi.dev/ergon/lang/terraform/release"
)

// The spellings of Terraform and of its toolchain in configuration. A repository names them, so
// the test pins them.
const (
	pinnedLanguage  workspace.Language  = "terraform"
	pinnedToolchain workspace.Toolchain = "terraform"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

// repository is the access to a repository of Terraform whose lock names it demo, and which has no
// tags.
var repository = terraform.Repository{
	Tags: func(context.Context, string) (map[string]string, error) { return map[string]string{}, nil },
	Answers: func(context.Context, string) (language.Answers, error) {
		return language.Answers{Name: "demo", Languages: []workspace.Language{pinnedLanguage}}, nil
	},
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Terraform with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, terraform.Register(&c, repository), "Register of Terraform")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of Terraform, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, terraform.Register(&c, repository), "Register of Terraform")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Terraform")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("adds the versioner and the tagger of the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, terraform.Register(&c, repository), "Register of Terraform")
			versioner, ok := language.ToolchainRole[language.Versioner](&c, pinnedToolchain)
			assert.True(t, ok, "the versioner of Terraform")
			assert.Equal(t, versioner, language.Versioner(release.Versioner{}), "the versioner")
			tagger, ok := language.ToolchainRole[language.Tagger](&c, pinnedToolchain)
			assert.True(t, ok, "the tagger of Terraform")
			assert.Equal(t, tagger, language.Tagger(release.Tagger{}), "the tagger")
		})

		t.Run("adds a toolchain that records its version in its changelog", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, terraform.Register(&c, repository), "Register of Terraform")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the toolchain of Terraform")
			assert.True(t, toolchain.ChangelogVersion, "ChangelogVersion")
		})

		t.Run("adds a discovery that reads the repository", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, terraform.Register(&c, repository), "Register of Terraform")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the toolchain of Terraform")
			assert.NotNil(t, toolchain.Discover, "Discover")
			pkgs, err := toolchain.Discover(t.Context(), t.TempDir())
			assert.NoError(t, err, "Discover")
			assert.Equal(t, pkgs, []workspace.Package{{Name: "demo", Toolchain: pinnedToolchain, Dir: "."}},
				"the packages")
		})

		incomplete := []struct {
			name string
			give terraform.Repository
		}{
			{name: "returns ErrRepository for a repository without Tags", give: terraform.Repository{
				Answers: repository.Answers,
			}},
			{name: "returns ErrRepository for a repository without Answers", give: terraform.Repository{
				Tags: repository.Tags,
			}},
		}
		for _, tt := range incomplete {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var c language.Catalog
				assert.ErrorIs(t, terraform.Register(&c, tt.give), terraform.ErrRepository, "Register of Terraform")
				assert.Empty(t, slices.Collect(c.Toolchains()), "the toolchains of the catalog")
			})
		}

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of Terraform")
			assert.ErrorIs(t, terraform.Register(&c, repository), language.ErrRegistered, "Register of Terraform")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Terraform with other")
			assert.ErrorIs(t, terraform.Register(&c, repository), language.ErrRegistered, "Register of Terraform")
		})
	})
}
