// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language_test

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// releaser implements every release role, and tells two releasers apart by its name.
type releaser struct {
	// name tells two releasers of a case apart.
	name string
}

var (
	_ language.Versioner = releaser{}
	_ language.Tagger    = releaser{}
	_ language.Packer    = releaser{}
	_ language.Publisher = releaser{}
	_ language.Locker    = releaser{}
)

// Resolve returns ResolutionSelected.
func (releaser) Resolve(string, version.Version) (language.Resolution, error) {
	return language.ResolutionSelected, nil
}

// Rewrite returns req.
func (releaser) Rewrite(req string, _ version.Version) (string, error) {
	return req, nil
}

// Validate returns nil.
func (releaser) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply changes no path.
func (releaser) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return nil, nil
}

// Tag returns the name of p, an at sign and v.
func (releaser) Tag(p *workspace.Package, v version.Version) string {
	return p.Name + "@" + v.String()
}

// Pack builds nothing.
func (releaser) Pack(context.Context, string, []workspace.Package, string) error {
	return nil
}

// Published reports false.
func (releaser) Published(context.Context, *workspace.Package) (bool, error) {
	return false, nil
}

// Publish uploads nothing.
func (releaser) Publish(context.Context, string, []workspace.Package) error {
	return nil
}

// Stale returns nil.
func (releaser) Stale(context.Context, string, []workspace.Package) ([]string, error) {
	return nil, nil
}

// Lock returns nil.
func (releaser) Lock(context.Context, string, []workspace.Package) ([]string, error) {
	return nil, nil
}

func TestRelease(t *testing.T) {
	t.Parallel()

	t.Run("Resolution", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give language.Resolution
				want bool
			}{
				{name: "reports false for the zero value", give: 0, want: false},
				{name: "reports true for ResolutionExcluded", give: language.ResolutionExcluded, want: true},
				{name: "reports true for ResolutionPinned", give: language.ResolutionPinned, want: true},
				{name: "reports true for ResolutionSelected", give: language.ResolutionSelected, want: true},
				{name: "reports false for a value after ResolutionSelected", give: language.ResolutionSelected + 1},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
		})
	})

	t.Run("Edit", func(t *testing.T) {
		t.Parallel()

		t.Run("changes no requirement in its zero value", func(t *testing.T) {
			t.Parallel()
			var e language.Edit
			assert.Nil(t, e.Requirements, "the requirements of the zero value")
			assert.True(t, e.Version.IsZero(), "the version of the zero value")
		})
	})

	t.Run("Versioner", func(t *testing.T) {
		t.Parallel()

		t.Run("is a role that a toolchain registers", func(t *testing.T) {
			t.Parallel()
			got := registeredRole[language.Versioner](t)
			assert.Equal(t, got, language.Versioner(releaser{name: "first"}), "the Versioner of the toolchain")
		})
	})

	t.Run("Tagger", func(t *testing.T) {
		t.Parallel()

		t.Run("is a role that a toolchain registers", func(t *testing.T) {
			t.Parallel()
			got := registeredRole[language.Tagger](t)
			assert.Equal(t, got, language.Tagger(releaser{name: "first"}), "the Tagger of the toolchain")
		})
	})

	t.Run("Packer", func(t *testing.T) {
		t.Parallel()

		t.Run("is a role that a toolchain registers", func(t *testing.T) {
			t.Parallel()
			got := registeredRole[language.Packer](t)
			assert.Equal(t, got, language.Packer(releaser{name: "first"}), "the Packer of the toolchain")
		})
	})

	t.Run("Publisher", func(t *testing.T) {
		t.Parallel()

		t.Run("is a role that a toolchain registers", func(t *testing.T) {
			t.Parallel()
			got := registeredRole[language.Publisher](t)
			assert.Equal(t, got, language.Publisher(releaser{name: "first"}), "the Publisher of the toolchain")
		})
	})

	t.Run("Locker", func(t *testing.T) {
		t.Parallel()

		t.Run("is a role that a toolchain registers", func(t *testing.T) {
			t.Parallel()
			got := registeredRole[language.Locker](t)
			assert.Equal(t, got, language.Locker(releaser{name: "first"}), "the Locker of the toolchain")
		})
	})
}

// registeredRole registers a toolchain with two releasers and returns its first role R.
func registeredRole[R any](tb testing.TB) R {
	tb.Helper()
	var c language.Catalog
	assert.NoError(tb, language.RegisterToolchain(&c, language.Toolchain{Name: tool},
		releaser{name: "first"}, releaser{name: "second"}), "RegisterToolchain with two releasers")
	role, ok := language.ToolchainRole[R](&c, tool)
	assert.True(tb, ok, "ToolchainRole of tool")
	return role
}
