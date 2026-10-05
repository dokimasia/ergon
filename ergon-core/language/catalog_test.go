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
}

// registered returns a catalog with the toolchain tool and the languages alpha and beta, which
// name it, registered in that order.
func registered(tb testing.TB) *language.Catalog {
	tb.Helper()
	c := new(language.Catalog)
	assert.NoError(tb, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain of tool")
	assert.NoError(tb, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register of alpha")
	assert.NoError(tb, language.Register(c, language.Declaration{Name: beta, Toolchain: tool}), "Register of beta")
	return c
}
