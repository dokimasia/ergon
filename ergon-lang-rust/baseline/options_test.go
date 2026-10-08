// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/rust/baseline"
)

// setupRust is the pin of setup-rust-toolchain at the baseline.
var setupRust = workflow.Action{
	Uses:    "actions-rust-lang/setup-rust-toolchain",
	Commit:  "ecabd13d1c56bd1345c230e542e9144811ad706f",
	Release: "v2.0.0",
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that Rust does not have", func(t *testing.T) {
			t.Parallel()
			o := rustOptions()
			o.Check = option.Check{option.StepFuzz}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job, the analysis and the updates of Rust at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, rustOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-rust",
					Name:        "Rust",
					Permissions: map[string]string{"contents": "read"},
					Setup: &workflow.Setup{
						Files:    "rust-toolchain.toml",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{{
							Name: "Set up Rust",
							Uses: setupRust,
							With: map[string]string{"components": "rustfmt, clippy"},
						}},
						Timeout: 30,
					},
					Tools: true,
					Steps: []workflow.Step{{Name: "Check Rust", Run: []string{"make check-rust"}}},
				}},
				CodeQL: []workflow.CodeQL{
					{Language: "rust", Name: "Rust", BuildMode: "none", Files: "rust-toolchain.toml", Timeout: 30},
				},
				Updates: []workflow.Update{{Ecosystem: "cargo", Directories: []string{"/"}}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := rustOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("installs the toolchain of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := rustOptions()
			o.CI.Versions = []string{"stable", "1.90"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Steps[0].With, map[string]string{
				"components": "rustfmt, clippy",
				"toolchain":  "${{ matrix.version }}",
			}, "the inputs of setup-rust-toolchain")
		})
	})
}

// rustOptions returns new options of the section rust at the baseline.
func rustOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
