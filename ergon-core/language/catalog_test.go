// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// The toolchain and the languages of the cases. They follow the rule of a valid name and belong
// to no language module of ergon.
const (
	tool  workspace.Toolchain = "tool"
	alpha workspace.Language  = "alpha"
	beta  workspace.Language  = "beta"
	gamma workspace.Language  = "gamma"
)

// initializer is an Initializer of the cases, told apart by its name.
type initializer struct {
	// name tells two initializers of a case apart.
	name string
}

// Files returns no file.
func (initializer) Files(*language.Answers) ([]language.File, error) {
	return nil, nil
}

func TestCatalog(t *testing.T) {
	t.Parallel()

	t.Run("Languages", func(t *testing.T) {
		t.Parallel()

		t.Run("yields the languages in the order of registration", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, slices.Collect(registered(t).Languages()), []language.Declaration{
				{Name: alpha, Toolchain: tool},
				{Name: beta, Toolchain: tool},
			}, "the languages of the catalog")
		})

		t.Run("yields nothing for the zero value", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.Empty(t, slices.Collect(c.Languages()), "the languages of the zero value")
		})

		t.Run("stops when the loop breaks", func(t *testing.T) {
			t.Parallel()
			var first []language.Declaration
			for d := range registered(t).Languages() {
				first = append(first, d)
				break
			}
			assert.Equal(t, first, []language.Declaration{{Name: alpha, Toolchain: tool}},
				"the languages before the break")
		})
	})

	t.Run("Language", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the declaration of a registered language", func(t *testing.T) {
			t.Parallel()
			d, ok := registered(t).Language(beta)
			assert.True(t, ok, "Language of beta")
			assert.Equal(t, d, language.Declaration{Name: beta, Toolchain: tool}, "the declaration of beta")
		})

		t.Run("reports false for a language that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			_, ok := registered(t).Language(gamma)
			assert.False(t, ok, "Language of gamma")
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a registered toolchain", func(t *testing.T) {
			t.Parallel()
			toolchain, ok := registered(t).Toolchain(tool)
			assert.True(t, ok, "Toolchain of tool")
			assert.Equal(t, toolchain, language.Toolchain{Name: tool}, "the toolchain tool")
		})

		t.Run("reports false for a toolchain that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			_, ok := registered(t).Toolchain("other")
			assert.False(t, ok, "Toolchain of other")
		})
	})

	t.Run("RegisterToolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("adds a toolchain that a language can name", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: tool}),
				"RegisterToolchain of tool")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: alpha, Toolchain: tool}),
				"Register of a language of tool")
		})

		t.Run("returns ErrUnknownRole for a role that implements no role interface", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			err := language.RegisterToolchain(&c, language.Toolchain{Name: tool}, 42)
			assert.ErrorIs(t, err, language.ErrUnknownRole, "RegisterToolchain of tool with an int role")
			_, ok := c.Toolchain(tool)
			assert.False(t, ok, "Toolchain of tool after the error")
		})

		t.Run("returns ErrInvalidName for an invalid name", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, language.RegisterToolchain(&c, language.Toolchain{Name: "Tool"}),
				language.ErrInvalidName, "RegisterToolchain of Tool")
		})

		t.Run("returns ErrRegistered for a registered name", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.RegisterToolchain(registered(t), language.Toolchain{Name: tool}),
				language.ErrRegistered, "a second RegisterToolchain of tool")
		})
	})

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrInvalidName for an invalid name", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.Register(registered(t), language.Declaration{Name: "", Toolchain: tool}),
				language.ErrInvalidName, "Register of the empty name")
		})

		t.Run("returns ErrRegistered for a registered name", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.Register(registered(t), language.Declaration{Name: beta, Toolchain: tool}),
				language.ErrRegistered, "a second Register of beta")
		})

		t.Run("returns ErrUnknownToolchain for a toolchain that is not registered", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.Register(registered(t), language.Declaration{Name: gamma, Toolchain: "other"}),
				language.ErrUnknownToolchain, "Register of a language of another toolchain")
		})

		t.Run("returns ErrUnknownRole for a role that implements no role interface", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.Register(registered(t), language.Declaration{Name: gamma, Toolchain: tool}, 42),
				language.ErrUnknownRole, "Register of gamma with an int role")
		})

		t.Run("returns ErrUnknownRole for a nil role", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, language.Register(registered(t), language.Declaration{Name: gamma, Toolchain: tool}, nil),
				language.ErrUnknownRole, "Register of gamma with a nil role")
		})

		t.Run("leaves the catalog unchanged when it returns an error", func(t *testing.T) {
			t.Parallel()
			c := registered(t)
			assert.HasError(t, language.Register(c, language.Declaration{Name: gamma, Toolchain: "other"}),
				"Register of a language of another toolchain")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: alpha, Toolchain: tool},
				{Name: beta, Toolchain: tool},
			}, "the languages of the catalog")
		})
	})

	t.Run("Role", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the first role that implements the type", func(t *testing.T) {
			t.Parallel()
			c := registered(t)
			assert.NoError(t, language.Register(c, language.Declaration{Name: gamma, Toolchain: tool},
				initializer{name: "first"}, initializer{name: "second"}), "Register of gamma with two roles")
			role, ok := language.Role[language.Initializer](c, gamma)
			assert.True(t, ok, "Role of gamma")
			assert.Equal(t, role, language.Initializer(initializer{name: "first"}), "the role of gamma")
		})

		t.Run("reports false for a language without such a role", func(t *testing.T) {
			t.Parallel()
			_, ok := language.Role[language.Initializer](registered(t), alpha)
			assert.False(t, ok, "Role of alpha")
		})

		t.Run("reports false for a language that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			_, ok := language.Role[language.Initializer](registered(t), gamma)
			assert.False(t, ok, "Role of gamma")
		})
	})

	t.Run("ToolchainRole", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the first role that implements the type", func(t *testing.T) {
			t.Parallel()
			c := registered(t)
			assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: "shared"},
				initializer{name: "first"}, initializer{name: "second"}), "RegisterToolchain of shared with two roles")
			role, ok := language.ToolchainRole[language.Initializer](c, "shared")
			assert.True(t, ok, "ToolchainRole of shared")
			assert.Equal(t, role, language.Initializer(initializer{name: "first"}), "the role of shared")
		})

		t.Run("reports false for a toolchain without such a role", func(t *testing.T) {
			t.Parallel()
			_, ok := language.ToolchainRole[language.Initializer](registered(t), tool)
			assert.False(t, ok, "ToolchainRole of tool")
		})

		t.Run("reports false for a toolchain that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			_, ok := language.ToolchainRole[language.Initializer](registered(t), "other")
			assert.False(t, ok, "ToolchainRole of other")
		})
	})
}

// registered returns a catalog with the toolchain tool and the languages alpha and beta, which
// name it, registered in that order without roles.
func registered(tb testing.TB) *language.Catalog {
	tb.Helper()
	c := new(language.Catalog)
	assert.NoError(tb, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain of tool")
	assert.NoError(tb, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register of alpha")
	assert.NoError(tb, language.Register(c, language.Declaration{Name: beta, Toolchain: tool}), "Register of beta")
	return c
}
