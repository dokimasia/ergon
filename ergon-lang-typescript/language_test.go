// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package typescript_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/typescript"
)

// The spellings of TypeScript and of the js toolchain in configuration. A repository names them,
// so the test pins them.
const (
	pinnedLanguage  workspace.Language  = "typescript"
	pinnedToolchain workspace.Toolchain = "js"
)

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds TypeScript with the js toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c), "Register of JavaScript")
			assert.NoError(t, typescript.Register(&c), "Register of TypeScript")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: javascript.Language, Toolchain: pinnedToolchain},
				{Name: pinnedLanguage, Toolchain: pinnedToolchain},
			}, "the languages of the catalog")
		})

		t.Run("returns ErrUnknownToolchain for a catalog without the js toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, typescript.Register(&c), language.ErrUnknownToolchain, "Register of TypeScript")
		})

		t.Run("returns ErrRegistered for a catalog that has TypeScript", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c), "Register of JavaScript")
			assert.NoError(t, typescript.Register(&c), "the first Register of TypeScript")
			assert.ErrorIs(t, typescript.Register(&c), language.ErrRegistered, "the second Register of TypeScript")
		})
	})
}
