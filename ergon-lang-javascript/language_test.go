// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package javascript_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

// The spellings of JavaScript and of the js toolchain in configuration. A repository names them,
// so the test pins them.
const (
	pinnedLanguage  workspace.Language  = "javascript"
	pinnedToolchain workspace.Toolchain = "js"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds JavaScript with the js toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c), "Register of JavaScript")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of the js toolchain, whose section is named after the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c), "Register of JavaScript")
			producer, ok := language.ToolchainRole[language.Producer](&c, pinnedToolchain)
			assert.True(t, ok, "the producer of js")
			assert.Equal(t, producer, language.Producer(baseline.Toolchain{}), "the producer")
			assert.Equal(t, baseline.ToolchainName, string(pinnedToolchain), "the name of the section")
		})

		t.Run("adds the producer of JavaScript, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c), "Register of JavaScript")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of JavaScript")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of js")
			assert.ErrorIs(t, javascript.Register(&c), language.ErrRegistered, "Register of JavaScript")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of JavaScript with other")
			assert.ErrorIs(t, javascript.Register(&c), language.ErrRegistered, "Register of JavaScript")
		})
	})
}
