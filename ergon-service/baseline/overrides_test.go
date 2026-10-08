// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.dokimi.dev/ergon/service/pin"
)

// toolingSection is the name of the producer tooling and of its section of .ergon.yaml.
const toolingSection = "tooling"

// The versions of the linter of the producer tooling: an earlier baseline and the baseline.
const (
	earlierLint option.Module = "golang.org/x/vuln/cmd/govulncheck@v1.7.0"
	currentLint option.Module = "golang.org/x/vuln/cmd/govulncheck@v1.8.0"
)

// pinned are the options of the producer tooling, with a pin and a plain option.
type pinned struct {
	// Lint is a Go module.
	Lint option.Module `yaml:"lint" doc:"The linter."`

	// Paths is no pin.
	Paths option.Paths `yaml:"paths" doc:"The paths."`
}

// Validate returns nil.
func (*pinned) Validate() error {
	return nil
}

// tooling is a base producer of the cases without templates, whose options have a pin.
type tooling struct {
	producer

	// lint is the linter of the baseline.
	lint option.Module
}

// Options returns the options at the baseline.
func (t tooling) Options() language.Options {
	return &pinned{Lint: t.lint, Paths: option.Paths{"./..."}}
}

// sourced is options with a version whose source tag names no registry.
type sourced struct {
	// Make has an invalid source tag.
	Make option.Version `yaml:"make" doc:"The version of make." source:"make"`
}

// Validate returns nil.
func (*sourced) Validate() error {
	return nil
}

// unsourced is a base producer of the cases without templates, whose options have an invalid source
// tag.
type unsourced struct {
	producer
}

// Options returns the options at the baseline.
func (unsourced) Options() language.Options {
	return &sourced{Make: "4.4.1"}
}

func TestOverrides(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("Overrides", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a pin that .ergon.yaml sets to another version", func(t *testing.T) {
				t.Parallel()
				r, root := tooled(t, currentLint)
				put(t, root, config, "tooling:\n  lint: "+string(earlierLint)+"\n")
				got, err := r.Overrides()
				assert.NoError(t, err, "Overrides")
				assert.Equal(t, got, []baseline.Override{
					{Key: "tooling.lint", Version: "v1.7.0", Baseline: "v1.8.0"},
				}, "the overrides")
			})

			t.Run("returns no override for .ergon.yaml at the baseline", func(t *testing.T) {
				t.Parallel()
				r, _ := tooled(t, currentLint)
				got, err := r.Overrides()
				assert.NoError(t, err, "Overrides")
				assert.Empty(t, got, "the overrides")
			})

			t.Run("returns no override for a pin at the value that the lock records", func(t *testing.T) {
				t.Parallel()
				_, root := tooled(t, earlierLint)
				r, err := baseline.Open(root, catalog(t), version, common("hello"),
					baseline.Producer{Name: toolingSection, Producer: tooling{lint: currentLint}})
				assert.NoError(t, err, "Open of the later baseline")
				got, err := r.Overrides()
				assert.NoError(t, err, "Overrides")
				assert.Empty(t, got, "the overrides")
			})

			t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
				t.Parallel()
				_, err := repository(t, directory(t)).Overrides()
				assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Overrides")
			})

			t.Run("returns ErrUnsupported for a lock with a language that lost its producer", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				c := new(language.Catalog)
				assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain")
				assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register")
				r, err := baseline.Open(root, c, version, common("hello"))
				assert.NoError(t, err, "Open")
				_, err = r.Overrides()
				assert.ErrorIs(t, err, baseline.ErrUnsupported, "Overrides")
			})

			t.Run(
				"returns ErrInvalid of the options for .ergon.yaml that the producers do not accept",
				func(t *testing.T) {
					t.Parallel()
					r, root := tooled(t, currentLint)
					put(t, root, config, "tooling:\n  lint: [a]\n")
					_, err := r.Overrides()
					assert.ErrorIs(t, err, options.ErrInvalid, "Overrides")
				},
			)

			t.Run("returns ErrSource of pin for a baseline with an invalid source tag", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				r, err := baseline.Open(root, catalog(t), version,
					baseline.Producer{Name: "sourced", Producer: unsourced{producer{templates: fstest.MapFS{}}}})
				assert.NoError(t, err, "Open")
				a := answers()
				a.Languages = []workspace.Language{}
				_, err = r.New(a, baseline.Options{})
				assert.NoError(t, err, "New")
				_, err = r.Overrides()
				assert.ErrorIs(t, err, pin.ErrSource, "Overrides")
			})
		})
	})
}

// tooled returns a repository and its directory after New with the answers of the cases and the
// language beta, which has no options. Its base producers are common and tooling with lint at the
// baseline.
func tooled(t *testing.T, lint option.Module) (*baseline.Repository, *os.Root) {
	t.Helper()
	root := directory(t)
	r, err := baseline.Open(
		root,
		catalog(t),
		version,
		common("hello"),
		baseline.Producer{
			Name:     toolingSection,
			Producer: tooling{lint: lint, producer: producer{templates: fstest.MapFS{}}},
		},
	)
	assert.NoError(t, err, "Open with the producer tooling")
	a := answers()
	a.Languages = []workspace.Language{alpha, beta}
	_, err = r.New(a, baseline.Options{})
	assert.NoError(t, err, "New of the repository")
	return r, root
}
