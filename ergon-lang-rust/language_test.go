// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package rust_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/rust/baseline"
)

// The spellings of Rust and of its toolchain in configuration. A repository names them, so the
// test pins them.
const (
	pinnedLanguage  workspace.Language  = "rust"
	pinnedToolchain workspace.Toolchain = "rust"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Rust with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, rust.Register(&c), "Register of Rust")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of Rust, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, rust.Register(&c), "Register of Rust")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Rust")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of Rust")
			assert.ErrorIs(t, rust.Register(&c), language.ErrRegistered, "Register of Rust")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Rust with other")
			assert.ErrorIs(t, rust.Register(&c), language.ErrRegistered, "Register of Rust")
		})
	})
}
