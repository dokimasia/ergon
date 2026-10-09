// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"context"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/lang/go/release"
)

// The spellings of Go and of its toolchain in configuration. A repository names them, so the test
// pins them.
const (
	pinnedLanguage  workspace.Language  = "go"
	pinnedToolchain workspace.Toolchain = "go"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

// git is the access to a repository of the cases, which has no tags and no tree.
var git = golang.Git{
	Tags:     func(context.Context, string) (map[string]string, error) { return nil, nil },
	Snapshot: func(context.Context, string) (string, error) { return "", nil },
	Head:     func(context.Context, string) (string, error) { return "", nil },
	LightTag: func(context.Context, string, string, string) error { return nil },
}

// tools runs no tool.
func tools(context.Context, string, string, []string, []string) error {
	return nil
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Go with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, golang.Register(&c, git, tools), "Register of Go")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of Go, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, golang.Register(&c, git, tools), "Register of Go")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Go")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("adds a toolchain that records the version of a module in its changelog", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, golang.Register(&c, git, tools), "Register of Go")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the toolchain of Go")
			assert.True(t, toolchain.ChangelogVersion, "ChangelogVersion")
			assert.NotNil(t, toolchain.Discover, "Discover")
		})

		t.Run("adds the versioner, the tagger and the packer of the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, golang.Register(&c, git, tools), "Register of Go")
			_, ok := language.ToolchainRole[language.Versioner](&c, pinnedToolchain)
			assert.True(t, ok, "the versioner of Go")
			tagger, ok := language.ToolchainRole[language.Tagger](&c, pinnedToolchain)
			assert.True(t, ok, "the tagger of Go")
			assert.Equal(t, tagger, language.Tagger(release.Tagger{}), "the tagger")
			packer, ok := language.ToolchainRole[language.Packer](&c, pinnedToolchain)
			assert.True(t, ok, "the packer of Go")
			_, isGo := packer.(release.Packer)
			assert.True(t, isGo, "the packer is the packer of Go")
		})

		gits := []struct {
			give  golang.Git
			name  string
			tools golang.Tools
		}{
			{
				name: "returns ErrGit for a git without Tags", tools: tools,
				give: golang.Git{Snapshot: git.Snapshot, Head: git.Head, LightTag: git.LightTag},
			},
			{
				name: "returns ErrGit for a git without Snapshot", tools: tools,
				give: golang.Git{Tags: git.Tags, Head: git.Head, LightTag: git.LightTag},
			},
			{
				name: "returns ErrGit for a git without Head", tools: tools,
				give: golang.Git{Tags: git.Tags, Snapshot: git.Snapshot, LightTag: git.LightTag},
			},
			{
				name: "returns ErrGit for a git without LightTag", tools: tools,
				give: golang.Git{Tags: git.Tags, Snapshot: git.Snapshot, Head: git.Head},
			},
			{name: "returns ErrGit without the tools", give: git},
		}
		for _, tt := range gits {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var c language.Catalog
				assert.ErrorIs(t, golang.Register(&c, tt.give, tt.tools), golang.ErrGit, "Register of Go")
			})
		}

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of Go")
			assert.ErrorIs(t, golang.Register(&c, git, tools), language.ErrRegistered, "Register of Go")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Go with other")
			assert.ErrorIs(t, golang.Register(&c, git, tools), language.ErrRegistered, "Register of Go")
		})
	})
}
