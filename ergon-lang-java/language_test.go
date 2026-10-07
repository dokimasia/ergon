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
	"go.dokimi.dev/ergon/lang/java/baseline"
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

		t.Run("adds the producer of the jvm toolchain, whose section is named after the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			producer, ok := language.ToolchainRole[language.Producer](&c, pinnedToolchain)
			assert.True(t, ok, "the producer of jvm")
			assert.Equal(t, producer, language.Producer(baseline.Toolchain{}), "the producer")
			assert.Equal(t, baseline.ToolchainName, string(pinnedToolchain), "the name of the section")
		})

		t.Run("adds the producer of Java, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, java.Register(&c), "Register of Java")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Java")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
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
