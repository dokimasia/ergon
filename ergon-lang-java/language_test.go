// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package java_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/java"
)

// The spellings of Java and of the jvm toolchain in configuration. A repository names them, so
// the test pins them.
const (
	pinnedLanguage  workspace.Language  = "java"
	pinnedToolchain workspace.Toolchain = "jvm"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Java with the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the init role of the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			_, ok := language.ToolchainRole[language.Initializer](&c, pinnedToolchain)
			assert.True(t, ok, "the init role of jvm")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of jvm")
			assert.ErrorIs(t, java.Register(&c), language.ErrRegistered, "Register of Java")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Java with other")
			assert.ErrorIs(t, java.Register(&c), language.ErrRegistered, "Register of Java")
		})
	})
}
