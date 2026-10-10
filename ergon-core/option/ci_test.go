// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// none is the struct of the pins of a producer whose jobs run no action of their own.
type none = struct{}

// The steps of ci.steps of the cases: a command that installs a compiler, and a step with neither
// an action nor a command.
var (
	installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}
	emptyStep         = workflow.Step{Name: "Empty"}
)

func TestCI(t *testing.T) {
	t.Parallel()

	t.Run("Runners", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Runners
			}{
				{
					name: "returns nil for the runners of the baseline",
					give: option.Runners{"ubuntu-26.04", "macos-26", "windows-2025"},
				},
				{name: "returns nil for no runner", give: option.Runners{}},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Runners
			}{
				{name: "returns ErrInvalid for a runner with a space", give: option.Runners{"ubuntu 26.04"}},
				{name: "returns ErrInvalid for an empty runner", give: option.Runners{""}},
				{name: "returns ErrInvalid for a runner named twice", give: option.Runners{"macos-26", "macos-26"}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a timeout of a minute", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, option.CI[none]{Timeout: 1}.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a timeout of 0", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, option.CI[none]{}.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("RunnerCI", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.RunnerCI[none]
			}{
				{name: "returns nil for no runner", give: option.RunnerCI[none]{Timeout: 30}},
				{
					name: "returns nil for runners of Linux and macOS",
					give: option.RunnerCI[none]{Runners: []string{"ubuntu-26.04", "macos-26"}, Timeout: 30},
				},
				{
					name: "returns nil for a step that runs a command",
					give: option.RunnerCI[none]{Steps: []workflow.Step{installTypeScript}, Timeout: 30},
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.RunnerCI[none]
			}{
				{
					name: "returns ErrInvalid for a runner with a space",
					give: option.RunnerCI[none]{Runners: []string{"ubuntu 26.04"}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for a runner named twice",
					give: option.RunnerCI[none]{Runners: []string{"macos-26", "macos-26"}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for a step without an action and without a command",
					give: option.RunnerCI[none]{Steps: []workflow.Step{emptyStep}, Timeout: 30},
				},
				{name: "returns ErrInvalid for a timeout of 0", give: option.RunnerCI[none]{}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}

			t.Run("returns an error that wraps ErrInvalidStep with the number of the step", func(t *testing.T) {
				t.Parallel()
				c := option.RunnerCI[none]{Steps: []workflow.Step{installTypeScript, emptyStep}, Timeout: 30}
				err := c.Validate()
				assert.ErrorIs(t, err, workflow.ErrInvalidStep, "the error of the step")
				assert.Contains(t, err.Error(), "step 2", "the number of the step")
			})
		})
	})

	t.Run("MatrixCI", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.MatrixCI[none]
			}{
				{name: "returns nil for no runner and no version", give: option.MatrixCI[none]{Timeout: 30}},
				{
					name: "returns nil for versions of Node with a slash and a star",
					give: option.MatrixCI[none]{Versions: []string{"22", "lts/*"}, Timeout: 30},
				},
				{
					name: "returns nil for a step that runs a command",
					give: option.MatrixCI[none]{Steps: []workflow.Step{installTypeScript}, Timeout: 30},
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.MatrixCI[none]
			}{
				{
					name: "returns ErrInvalid for a runner with a colon",
					give: option.MatrixCI[none]{Runners: []string{"a:b"}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for an empty version",
					give: option.MatrixCI[none]{Versions: []string{""}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for a version that spans lines",
					give: option.MatrixCI[none]{Versions: []string{"1.27\n1.26"}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for a version named twice",
					give: option.MatrixCI[none]{Versions: []string{"1.27", "1.27"}, Timeout: 30},
				},
				{
					name: "returns ErrInvalid for a step without an action and without a command",
					give: option.MatrixCI[none]{Steps: []workflow.Step{emptyStep}, Timeout: 30},
				},
				{name: "returns ErrInvalid for a timeout of 0", give: option.MatrixCI[none]{}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}

			t.Run("returns an error that wraps ErrInvalidStep with the number of the step", func(t *testing.T) {
				t.Parallel()
				c := option.MatrixCI[none]{Steps: []workflow.Step{installTypeScript, emptyStep}, Timeout: 30}
				err := c.Validate()
				assert.ErrorIs(t, err, workflow.ErrInvalidStep, "the error of the step")
				assert.Contains(t, err.Error(), "step 2", "the number of the step")
			})
		})
	})
}
