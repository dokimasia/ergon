// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/spec"
)

// name pins the name of the producer of spec.
const name = "spec"

// read is the permission of the job spec.
var read = map[string]string{"contents": "read"}

// The steps of the job spec of each role.
var (
	// checkCopies is the step of an implementation.
	checkCopies = workflow.Step{
		Name: "Check the copies of the definitions",
		Env:  map[string]string{"GITHUB_TOKEN": "${{ github.token }}"},
		Run:  []string{"ergon tool run spec.spec -- check --remote"},
	}

	// checkManifest is the step of a definition.
	checkManifest = workflow.Step{
		Name: "Check the manifest of the definition",
		Run:  []string{"ergon tool run spec.spec -- manifest --check"},
	}
)

// roled is the producer of spec whose options are the options at the baseline with the roles of the
// case.
type roled struct {
	spec.Producer

	// roles are the roles of the options.
	roles []spec.Role
}

// Options returns the options of spec at the baseline with the roles of r.
func (r roled) Options() language.Options {
	o, _ := r.Producer.Options().(*spec.Options)
	o.Roles = r.roles
	return o
}

func TestSpec(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is spec", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, spec.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name   string
				golden string
				give   []spec.Role
			}{
				{name: "renders no file of spec without a role", golden: "baseline", give: []spec.Role{}},
				{
					name:   "renders check-spec and spec.yml for an implementation",
					golden: "implementation",
					give:   []spec.Role{spec.Implementation},
				},
				{
					name:   "renders generate-spec and verify-generate-spec for a definition",
					golden: "definition",
					give:   []spec.Role{spec.Definition},
				},
				{
					name:   "renders the targets and the workflow of both roles",
					golden: "both",
					give:   []spec.Role{spec.Definition, spec.Implementation},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					p := baseline.Producer{Name: spec.Name, Producer: roled{roles: tt.give}}
					dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), p)
					baselinetest.Hygiene(t, dir)
					golden.MatchTree(t, tt.golden, os.DirFS(dir), golden.ShouldUpdate())
				})
			}
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns no role", func(t *testing.T) {
				t.Parallel()
				o, _ := spec.Producer{}.Options().(*spec.Options)
				assert.Empty(t, o.Roles, "the roles")
			})

			t.Run("returns valid options", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, spec.Producer{}.Options().Validate(), "Validate")
			})

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := spec.Producer{}.Options().(*spec.Options)
				second, _ := spec.Producer{}.Options().(*spec.Options)
				first.Tools.Spec.SHA256["linux/amd64"] = "changed"
				assert.NotEqual(t, second.Tools.Spec.SHA256["linux/amd64"], "changed", "the digest of the second value")
			})
		})

		t.Run("Data", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give language.Options
				want spec.Repository
			}{
				{name: "returns no role for the options at the baseline", give: spec.Producer{}.Options()},
				{
					name: "returns an implementation",
					give: roled{roles: []spec.Role{spec.Implementation}}.Options(),
					want: spec.Repository{Implementation: true},
				},
				{
					name: "returns a definition",
					give: roled{roles: []spec.Role{spec.Definition}}.Options(),
					want: spec.Repository{Definition: true},
				},
				{
					name: "returns both roles",
					give: roled{roles: []spec.Role{spec.Implementation, spec.Definition}}.Options(),
					want: spec.Repository{Implementation: true, Definition: true},
				},
				{name: "returns no role for options of another type", give: nil},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := spec.Producer{}.Data(baselinetest.Answers(), tt.give, &workflow.Contribution{})
					assert.NoError(t, err, "Data")
					assert.Equal(t, got, any(tt.want), "the data")
				})
			}
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give []spec.Role
				want []workflow.Step
			}{
				{
					name: "returns the job spec with the check of the copies for an implementation",
					give: []spec.Role{spec.Implementation},
					want: []workflow.Step{checkCopies},
				},
				{
					name: "returns the job spec with the check of the manifest for a definition",
					give: []spec.Role{spec.Definition},
					want: []workflow.Step{checkManifest},
				},
				{
					name: "returns the check of the copies before the check of the manifest for both roles",
					give: []spec.Role{spec.Definition, spec.Implementation},
					want: []workflow.Step{checkCopies, checkManifest},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got := spec.Producer{}.Contribution(roled{roles: tt.give}.Options())
					assert.Equal(t, got, workflow.Contribution{Jobs: []workflow.Job{{
						ID:          "spec",
						Name:        "Spec",
						Text:        true,
						Timeout:     10,
						Permissions: read,
						Ergon:       true,
						Tools:       true,
						Steps:       tt.want,
					}}}, "the contribution")
					assert.NoError(t, got.Validate(), "Validate of the contribution")
				})
			}

			t.Run("returns no job without a role", func(t *testing.T) {
				t.Parallel()
				got := spec.Producer{}.Contribution(spec.Producer{}.Options())
				assert.Equal(t, got, workflow.Contribution{}, "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, spec.Producer{}.Contribution(nil), workflow.Contribution{}, "the contribution")
			})

			t.Run("returns the job with the timeout of the options", func(t *testing.T) {
				t.Parallel()
				o, _ := roled{roles: []spec.Role{spec.Implementation}}.Options().(*spec.Options)
				o.CI.Timeout = 25
				got := spec.Producer{}.Contribution(o)
				assert.Length(t, got.Jobs, 1, "the jobs")
				assert.Equal(t, got.Jobs[0].Timeout, 25, "the timeout of spec")
			})
		})
	})
}
