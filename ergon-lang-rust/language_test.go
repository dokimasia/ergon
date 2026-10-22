// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package rust_test

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
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

// crate is the Cargo.toml of a crate of the cases.
const crate = "[package]\nname = \"probe\"\nversion = \"0.1.0\"\n"

// errPackage is the error of a run of cargo package of the cases that fails.
var errPackage = errors.New("cargo: the package failed")

// none does not run a program, and reads an environment without variables.
var none = rust.Tools{
	Command: func(context.Context, string, string, []string, []string) error { return nil },
	Getenv:  func(string) string { return "" },
}

// index is a transport that records the address of each request and answers it with 404, as the
// index of crates.io answers for a crate that crates.io does not have.
type index struct {
	urls []string
}

// RoundTrip records the address of r and answers it with 404.
func (f *index) RoundTrip(r *http.Request) (*http.Response, error) {
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

		t.Run("adds Rust with its own toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, rust.Register(&c, none, &http.Client{}), "Register of Rust")
			assert.Equal(t, slices.Collect(c.Languages()),
				[]language.Declaration{{Name: pinnedLanguage, Toolchain: pinnedToolchain}},
				"the languages of the catalog")
		})

		t.Run("adds the producer of Rust, whose section is named after the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, rust.Register(&c, none, &http.Client{}), "Register of Rust")
			producer, ok := language.Role[language.Producer](&c, pinnedLanguage)
			assert.True(t, ok, "the producer of Rust")
			assert.Equal(t, producer, language.Producer(baseline.Producer{}), "the producer")
			assert.Equal(t, baseline.Name, string(pinnedLanguage), "the name of the section")
		})

		t.Run("adds the discovery of the crates of a Cargo workspace", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, rust.Register(&c, none, &http.Client{}), "Register of Rust")
			toolchain, ok := c.Toolchain(pinnedToolchain)
			assert.True(t, ok, "the toolchain of Rust")
			pkgs, err := toolchain.Discover(
				t.Context(),
				files.Workspace(t, files.Tree{"Cargo.toml": files.Text(crate)}),
			)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, pkgs, []workspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Minor: 1}},
			}, "the packages")
		})

		t.Run("adds a versioner that refreshes Cargo.lock with cargo of the tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			var ran []string
			tools := none
			tools.Command = func(_ context.Context, _, name string, args, _ []string) error {
				ran = append(append(ran, name), args...)
				return nil
			}
			assert.NoError(t, rust.Register(&c, tools, &http.Client{}), "Register of Rust")
			v, ok := language.ToolchainRole[language.Versioner](&c, pinnedToolchain)
			assert.True(t, ok, "the versioner of Rust")
			root := files.Workspace(t, files.Tree{"Cargo.toml": files.Text(crate), "Cargo.lock": files.Text("lock\n")})
			_, err := v.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "probe", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, ran, []string{"cargo", "update", "--workspace"}, "the program and its arguments")
		})

		t.Run("adds a packer that packages with cargo of the tools", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			var ran []string
			tools := none
			tools.Command = func(_ context.Context, _, name string, args, _ []string) error {
				ran = append(append(ran, name), args...)
				return errPackage
			}
			assert.NoError(t, rust.Register(&c, tools, &http.Client{}), "Register of Rust")
			p, ok := language.ToolchainRole[language.Packer](&c, pinnedToolchain)
			assert.True(t, ok, "the packer of Rust")
			root := t.TempDir()
			err := p.Pack(t.Context(), root, []workspace.Package{
				{Name: "probe", Dir: ".", Version: version.Version{Minor: 2}},
			}, t.TempDir())
			assert.ErrorIs(t, err, errPackage, "Pack")
			assert.Equal(t, ran, []string{
				"cargo", "package", "--target-dir", filepath.Join(root, "target", "ergon"), "--package", "probe",
			}, "the program and its arguments")
		})

		t.Run("adds a publisher that reads the index of crates.io through the client", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			f := &index{}
			assert.NoError(t, rust.Register(&c, none, &http.Client{Transport: f}), "Register of Rust")
			p, ok := language.ToolchainRole[language.Publisher](&c, pinnedToolchain)
			assert.True(t, ok, "the publisher of Rust")
			got, err := p.Published(t.Context(), &workspace.Package{Name: "probe", Version: version.Version{Minor: 2}})
			assert.NoError(t, err, "Published")
			assert.False(t, got, "Published")
			assert.Equal(t, f.urls, []string{"https://index.crates.io/pr/ob/probe"}, "the requests")
		})

		tests := []struct {
			name      string
			giveTools rust.Tools
		}{
			{name: "returns ErrTools for tools without Command", giveTools: rust.Tools{Getenv: none.Getenv}},
			{name: "returns ErrTools for tools without Getenv", giveTools: rust.Tools{Command: none.Command}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var c language.Catalog
				assert.ErrorIs(t, rust.Register(&c, tt.giveTools, &http.Client{}), rust.ErrTools, "Register of Rust")
			})
		}

		t.Run("returns ErrClient for a nil client", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.ErrorIs(t, rust.Register(&c, none, nil), rust.ErrClient, "Register of Rust")
		})

		t.Run("returns ErrRegistered for a catalog that has the toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: pinnedToolchain}),
				"RegisterToolchain of the toolchain of Rust")
			assert.ErrorIs(t, rust.Register(&c, none, &http.Client{}), language.ErrRegistered, "Register of Rust")
		})

		t.Run("returns ErrRegistered for a catalog that has the language", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: other}),
				"RegisterToolchain of other")
			assert.NoError(t, language.Register(&c, language.Declaration{Name: pinnedLanguage, Toolchain: other}),
				"Register of Rust with other")
			assert.ErrorIs(t, rust.Register(&c, none, &http.Client{}), language.ErrRegistered, "Register of Rust")
		})
	})
}
