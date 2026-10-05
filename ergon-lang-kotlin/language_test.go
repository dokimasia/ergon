// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package kotlin_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/kotlin"
)

// The spellings of Kotlin and of the jvm toolchain in configuration. A repository names them, so
// the test pins them.
const (
	pinnedLanguage  workspace.Language  = "kotlin"
	pinnedToolchain workspace.Toolchain = "jvm"
)

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Kotlin with the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			assert.NoError(t, kotlin.Register(&c), "Register of Kotlin")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: java.Language, Toolchain: pinnedToolchain},
				{Name: pinnedLanguage, Toolchain: pinnedToolchain},
			}, "the languages of the catalog")
		})

		t.Run("returns ErrUnknownToolchain for a catalog without the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, kotlin.Register(&c), language.ErrUnknownToolchain, "Register of Kotlin")
		})

		t.Run("returns ErrRegistered for a catalog that has Kotlin", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			assert.NoError(t, kotlin.Register(&c), "the first Register of Kotlin")
			assert.ErrorIs(t, kotlin.Register(&c), language.ErrRegistered, "the second Register of Kotlin")
		})
	})
}
