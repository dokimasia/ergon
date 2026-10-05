// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds every language with the toolchain that builds it", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, app.Register(&c), "Register of every language module")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: csharp.Language, Toolchain: csharp.Toolchain},
				{Name: java.Language, Toolchain: java.Toolchain},
				{Name: kotlin.Language, Toolchain: java.Toolchain},
				{Name: php.Language, Toolchain: php.Toolchain},
				{Name: javascript.Language, Toolchain: javascript.Toolchain},
				{Name: typescript.Language, Toolchain: javascript.Toolchain},
				{Name: golang.Language, Toolchain: golang.Toolchain},
				{Name: python.Language, Toolchain: python.Toolchain},
				{Name: rust.Language, Toolchain: rust.Toolchain},
				{Name: terraform.Language, Toolchain: terraform.Toolchain},
				{Name: bash.Language, Toolchain: bash.Toolchain},
			}, "the languages of the catalog")
		})

		t.Run("returns ErrRegistered for a catalog that has a language of ergon", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, csharp.Register(&c), "Register of C#")
			assert.ErrorIs(t, app.Register(&c), language.ErrRegistered, "Register of every language module")
		})
	})
}
