// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.dokimi.dev/ergon/service/pin"
)

// The pins of the options of the cases, one of each kind.
var (
	lint   = option.Module("golang.org/x/vuln/cmd/govulncheck@v1.8.0")
	ruff   = option.PyPI("ruff@0.16.10")
	biome  = option.NPM("@biomejs/biome@2.5.15")
	audit  = option.Crate("cargo-audit@0.22.2")
	ktlint = option.Maven("com.pinterest.ktlint:ktlint-cli@1.8.0")
	stan   = option.Composer("phpstan/phpstan@2.2.17")
	binary = tool{Version: "0.12.0", SHA256: map[option.Platform]string{option.LinuxAMD64: linuxDigest}, Name: program}
	setup  = workflow.Action{Uses: actionRepo, Commit: actionCommit, Release: "v7.0.0"}
	codeql = workflow.Action{
		Uses: "github/codeql-action/init", Commit: "5b6e617dc0241b2d60c2bccea90c56b67eceb797", Release: "v4.38.2",
	}
)

// demoTools are the tools of the options of the cases.
type demoTools struct {
	// Lint is a Go module.
	Lint option.Module `yaml:"lint"`

	// Ruff is a PyPI package.
	Ruff option.PyPI `yaml:"ruff"`

	// Biome is an npm package of a scope.
	Biome option.NPM `yaml:"biome"`

	// Audit is a crate.
	Audit option.Crate `yaml:"audit"`

	// Ktlint is a Maven artifact.
	Ktlint option.Maven `yaml:"ktlint"`

	// Stan is a Composer package.
	Stan option.Composer `yaml:"stan"`

	// Tool is a release binary.
	Tool tool `yaml:"tool"`
}

// demoActions are the actions of the options of the cases.
type demoActions struct {
	// Setup is an action at the root of its repository.
	Setup workflow.Action `yaml:"setup"`

	// Init is an action in a directory of its repository.
	Init workflow.Action `yaml:"init"`
}

// demo is options with a pin of every kind, a version without a source tag, and a plain option.
type demo struct {
	// Tools are the tools.
	Tools demoTools `yaml:"tools"`

	// Actions are the actions.
	Actions demoActions `yaml:"actions"`

	// Make is a package of Chocolatey.
	Make option.Version `yaml:"make" source:"chocolatey:make"`

	// Hooks is a version of the releases of a repository on GitHub.
	Hooks option.Version `yaml:"hooks" source:"github:pre-commit/pre-commit-hooks"`

	// Go is a version without a source tag, so it is no pin.
	Go option.Version `yaml:"go"`

	// Name is a plain option.
	Name string `yaml:"name"`
}

// Validate returns nil.
func (*demo) Validate() error {
	return nil
}

// Shared is a group of options that other options embed inline.
type Shared struct {
	// Lint is a Go module.
	Lint option.Module `yaml:"lint"`
}

// inlined is options that embed Shared inline.
type inlined struct {
	Shared `yaml:",inline"`
}

// Validate returns nil.
func (*inlined) Validate() error {
	return nil
}

// npmSource is options with a source tag of a registry that is neither Chocolatey nor GitHub, with
// a name that is a repository on GitHub.
type npmSource struct {
	// Version names npm as its source.
	Version option.Version `yaml:"version" source:"npm:example/tool"`
}

// Validate returns nil.
func (*npmSource) Validate() error {
	return nil
}

// emptySource is options with a source tag of Chocolatey without a package.
type emptySource struct {
	// Version names no package.
	Version option.Version `yaml:"version" source:"chocolatey:"`
}

// Validate returns nil.
func (*emptySource) Validate() error {
	return nil
}

// ownerSource is options with a source tag of GitHub that names an owner without a repository.
type ownerSource struct {
	// Version names an owner alone.
	Version option.Version `yaml:"version" source:"github:pre-commit"`
}

// Validate returns nil.
func (*ownerSource) Validate() error {
	return nil
}

// quiet is options without pins.
type quiet struct {
	// Name is a plain option.
	Name string `yaml:"name"`
}

// Validate returns nil.
func (*quiet) Validate() error {
	return nil
}

// flat is options that are a struct and not a pointer to one.
type flat struct{}

// Validate returns nil.
func (flat) Validate() error {
	return nil
}

func TestPin(t *testing.T) {
	t.Parallel()

	t.Run("Find", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pins of every kind in the order of the keys", func(t *testing.T) {
			t.Parallel()
			got, err := pin.Find("demo", &demo{
				Tools: demoTools{
					Lint: lint, Ruff: ruff, Biome: biome, Audit: audit, Ktlint: ktlint, Stan: stan, Tool: binary,
				},
				Actions: demoActions{Setup: setup, Init: codeql},
				Make:    "4.4.1",
				Hooks:   "v6.0.0",
				Go:      "1.27.1",
				Name:    "demo",
			})
			assert.NoError(t, err, "Find")
			assert.Equal(t, got, []pin.Pin{
				{
					Value: lint, Key: "demo.tools.lint", Name: "golang.org/x/vuln/cmd/govulncheck", Version: "v1.8.0",
					Field: []string{"Tools", "Lint"}, Kind: pin.KindModule,
				},
				{
					Value: ruff, Key: "demo.tools.ruff", Name: "ruff", Version: "0.16.10",
					Field: []string{"Tools", "Ruff"}, Kind: pin.KindPyPI,
				},
				{
					Value: biome, Key: "demo.tools.biome", Name: "@biomejs/biome", Version: "2.5.15",
					Field: []string{"Tools", "Biome"}, Kind: pin.KindNPM,
				},
				{
					Value: audit, Key: "demo.tools.audit", Name: "cargo-audit", Version: "0.22.2",
					Field: []string{"Tools", "Audit"}, Kind: pin.KindCrate,
				},
				{
					Value: ktlint, Key: "demo.tools.ktlint", Name: "com.pinterest.ktlint:ktlint-cli", Version: "1.8.0",
					Field: []string{"Tools", "Ktlint"}, Kind: pin.KindMaven,
				},
				{
					Value: stan, Key: "demo.tools.stan", Name: "phpstan/phpstan", Version: "2.2.17",
					Field: []string{"Tools", "Stan"}, Kind: pin.KindComposer,
				},
				{
					Value: binary, Key: "demo.tools.tool", Name: toolRepo, Version: "0.12.0",
					Field: []string{"Tools", "Tool"}, Kind: pin.KindBinary,
				},
				{
					Value: setup, Key: "demo.actions.setup", Name: actionRepo, Version: "v7.0.0",
					Field: []string{"Actions", "Setup"}, Kind: pin.KindAction,
				},
				{
					Value: codeql, Key: "demo.actions.init", Name: "github/codeql-action", Version: "v4.38.2",
					Field: []string{"Actions", "Init"}, Kind: pin.KindAction,
				},
				{
					Value: option.Version("4.4.1"), Key: "demo.make", Name: "make", Version: "4.4.1",
					Field: []string{"Make"}, Kind: pin.KindChocolatey,
				},
				{
					Value: option.Version("v6.0.0"), Key: "demo.hooks", Name: hooksRepo, Version: "v6.0.0",
					Field: []string{"Hooks"}, Kind: pin.KindGitHub,
				},
			}, "the pins")
		})

		t.Run("returns the key of a pin of an inline struct without the struct", func(t *testing.T) {
			t.Parallel()
			got, err := pin.Find("demo", &inlined{Shared{Lint: lint}})
			assert.NoError(t, err, "Find")
			assert.Equal(t, got, []pin.Pin{{
				Value: lint, Key: "demo.lint", Name: "golang.org/x/vuln/cmd/govulncheck", Version: "v1.8.0",
				Field: []string{"Shared", "Lint"}, Kind: pin.KindModule,
			}}, "the pins")
		})

		t.Run("returns no pins for options without pins", func(t *testing.T) {
			t.Parallel()
			got, err := pin.Find("demo", &quiet{Name: "demo"})
			assert.NoError(t, err, "Find")
			assert.Empty(t, got, "the pins")
		})

		sources := []struct {
			name string
			give language.Options
		}{
			{name: "returns ErrSource for a source tag of another registry", give: &npmSource{}},
			{name: "returns ErrSource for a source tag of Chocolatey without a package", give: &emptySource{}},
			{name: "returns ErrSource for a source tag of GitHub that is not owner/name", give: &ownerSource{}},
		}
		for _, tt := range sources {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := pin.Find("demo", tt.give)
				assert.ErrorIs(t, err, pin.ErrSource, "Find")
				assert.Contains(t, err.Error(), "demo.version", "the error")
			})
		}

		t.Run("returns ErrDefect for options that are not a pointer to a struct", func(t *testing.T) {
			t.Parallel()
			_, err := pin.Find("demo", flat{})
			assert.ErrorIs(t, err, options.ErrDefect, "Find")
		})
	})

	t.Run("Kind", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give pin.Kind
				want bool
			}{
				{name: "reports false for the zero kind", give: 0, want: false},
				{name: "reports true for KindModule", give: pin.KindModule, want: true},
				{name: "reports true for KindGitHub", give: pin.KindGitHub, want: true},
				{name: "reports false for a kind after KindGitHub", give: pin.KindGitHub + 1, want: false},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
		})
	})
}
