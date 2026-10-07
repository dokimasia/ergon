// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package php_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/php/baseline"
)

// The spellings of PHP and of its toolchain in configuration. A repository names them, so the test
// pins them.
const (
	pinnedLanguage  workspace.Language  = "php"
	pinnedToolchain workspace.Toolchain = "php"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds PHP with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, php.Register(&c), "Register of PHP")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of PHP, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, php.Register(&c), "Register of PHP")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of PHP")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of PHP")
			assert.ErrorIs(t, php.Register(&c), language.ErrRegistered, "Register of PHP")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of PHP with other")
			assert.ErrorIs(t, php.Register(&c), language.ErrRegistered, "Register of PHP")
		})
	})
}
