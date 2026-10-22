// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package javascript_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
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

// manifest is the package.json of a package of the cases.
const manifest = `{"name": "probe", "version": "0.1.0"}`

// errPack is the error of a pack of the cases that fails.
var errPack = errors.New("npm: the pack failed")

// none returns nil and does not run a program or a tool.
var none = javascript.Tools{
	Run:     func(context.Context, string, string, []string, []string) error { return nil },
	Command: func(context.Context, string, string, []string, []string) error { return nil },
}

// registry is a transport that records the address of each request and answers it with 404, as the
// npm registry answers for a version that it does not have.
type registry struct {
	urls []string
}

// RoundTrip records the address of r and answers it with 404.
func (f *registry) RoundTrip(r *http.Request) (*http.Response, error) {
	f.urls = append(f.urls, r.URL.String())
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Header:     http.Header{},
		Body:       http.NoBody,
		Request:    r,
	}, nil
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds JavaScript with the js toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c, none, &http.Client{}), "Register of JavaScript")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of the js toolchain, whose section is named after the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c, none, &http.Client{}), "Register of JavaScript")
			producer, ok := language.ToolchainRole[language.Producer](&c, pinnedToolchain)
			assert.True(t, ok, "the producer of js")
			assert.Equal(t, producer, language.Producer(baseline.Toolchain{}), "the producer")
			assert.Equal(t, baseline.ToolchainName, string(pinnedToolchain), "the name of the section")
		})

		t.Run("adds the producer of JavaScript, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c, none, &http.Client{}), "Register of JavaScript")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of JavaScript")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("adds the discovery of the packages of a workspace", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, javascript.Register(&c, none, &http.Client{}), "Register of JavaScript")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the js toolchain")
			pkgs, err := toolchain.Discover(
				t.Context(),
				files.Workspace(t, files.Tree{"package.json": files.Text(manifest)}),
			)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, pkgs, []workspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Minor: 1}},
			}, "the packages")
		})

		t.Run("adds a versioner that refreshes the lockfile with the program of the tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			var ran []string
			tools := none
			tools.Command = func(_ context.Context, _, name string, args, _ []string) error {
				ran = append(append(ran, name), args...)
				return nil
			}
			assert.NoError(t, javascript.Register(&c, tools, &http.Client{}), "Register of JavaScript")
			v, ok := language.ToolchainRole[language.Versioner](&c, pinnedToolchain)
			assert.True(t, ok, "the versioner of js")
			root := files.Workspace(t, files.Tree{
				"package.json": files.Text(manifest), "package-lock.json": files.Text("{}\n"),
			})
			_, err := v.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "probe", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, ran, []string{"npm", "install", "--package-lock-only"}, "the program and its arguments")
		})

		t.Run("adds a packer that packs with the program of the tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			var ran []string
			tools := none
			tools.Command = func(_ context.Context, _, name string, args, _ []string) error {
				ran = append(append(ran, name), args[0])
				return errPack
			}
			assert.NoError(t, javascript.Register(&c, tools, &http.Client{}), "Register of JavaScript")
			p, ok := language.ToolchainRole[language.Packer](&c, pinnedToolchain)
			assert.True(t, ok, "the packer of js")
			err := p.Pack(t.Context(), t.TempDir(), []workspace.Package{
				{Name: "probe", Dir: ".", Version: version.Version{Minor: 2}},
			}, t.TempDir())
			assert.ErrorIs(t, err, errPack, "Pack")
			assert.Equal(t, ran, []string{"npm", "pack"}, "the program and its command")
		})

		t.Run("adds a publisher that reads the npm registry through the client", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			f := &registry{}
			assert.NoError(t, javascript.Register(&c, none, &http.Client{Transport: f}), "Register of JavaScript")
			p, ok := language.ToolchainRole[language.Publisher](&c, pinnedToolchain)
			assert.True(t, ok, "the publisher of js")
			got, err := p.Published(t.Context(), &workspace.Package{Name: "probe", Version: version.Version{Minor: 2}})
			assert.NoError(t, err, "Published")
			assert.False(t, got, "Published")
			assert.Equal(t, f.urls, []string{"https://registry.npmjs.org/probe/0.2.0"}, "the requests")
		})

		tests := []struct {
			name      string
			giveTools javascript.Tools
		}{
			{name: "returns ErrTools for tools without Run", giveTools: javascript.Tools{Command: none.Command}},
			{name: "returns ErrTools for tools without Command", giveTools: javascript.Tools{Run: none.Run}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var c language.Catalog
				err := javascript.Register(&c, tt.giveTools, &http.Client{})
				assert.ErrorIs(t, err, javascript.ErrTools, "Register of JavaScript")
			})
		}

		t.Run("returns ErrClient for a nil client", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, javascript.Register(&c, none, nil), javascript.ErrClient, "Register of JavaScript")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of js")
			assert.ErrorIs(t, javascript.Register(&c, none, &http.Client{}), language.ErrRegistered,
				"Register of JavaScript")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of JavaScript with other")
			assert.ErrorIs(t, javascript.Register(&c, none, &http.Client{}), language.ErrRegistered,
				"Register of JavaScript")
		})
	})
}
