// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package python_test

import (
	"context"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/python"
	"go.dokimi.dev/ergon/lang/python/baseline"
)

// The spellings of Python and of its toolchain in configuration. A repository names them, so the
// test pins them.
const (
	pinnedLanguage  workspace.Language  = "python"
	pinnedToolchain workspace.Toolchain = "python"
)

// other is a toolchain of the cases, which belongs to no module of ergon.
const other workspace.Toolchain = "other"

// project is the pyproject.toml of a project of the cases.
const project = "[project]\nname = \"probe\"\nversion = \"0.1.0\"\n"

// none runs no tool and returns nil.
func none(context.Context, string, string, []string, []string) error {
	return nil
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds Python with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, python.Register(&c, none), "Register of Python")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of Python, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, python.Register(&c, none), "Register of Python")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Python")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("adds the discovery of the projects of a uv workspace", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, python.Register(&c, none), "Register of Python")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the toolchain of Python")
			pkgs, err := toolchain.Discover(
				t.Context(),
				files.Workspace(t, files.Tree{"pyproject.toml": files.Text(project)}),
			)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, pkgs, []workspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Minor: 1}},
			}, "the packages")
		})

		t.Run("adds a versioner that refreshes uv.lock with uv of the tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			var ran []string
			tools := func(_ context.Context, _, tool string, args, _ []string) error {
				ran = append(append(ran, tool), args...)
				return nil
			}
			assert.NoError(t, python.Register(&c, tools), "Register of Python")
			v, ok := language.ToolchainRole[language.Versioner](&c, pinnedToolchain)
			assert.True(t, ok, "the versioner of Python")
			root := files.Workspace(
				t,
				files.Tree{"pyproject.toml": files.Text(project), "uv.lock": files.Text("lock\n")},
			)
			_, err := v.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "probe", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, ran, []string{"uv", "lock"}, "the tool and its arguments")
		})

		t.Run("returns ErrTools for nil tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, python.Register(&c, nil), python.ErrTools, "Register of Python")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of Python")
			assert.ErrorIs(t, python.Register(&c, none), language.ErrRegistered, "Register of Python")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Python with other")
			assert.ErrorIs(t, python.Register(&c, none), language.ErrRegistered, "Register of Python")
		})
	})
}
