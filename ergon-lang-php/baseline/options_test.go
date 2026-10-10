// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/php/baseline"
)

// setupPHP is the pin of setup-php at the baseline.
var setupPHP = workflow.Action{
	Uses:    "shivammathur/setup-php",
	Commit:  "f3e473d116dcccaddc5834248c87452386958240",
	Release: "2.37.2",
}

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Tools", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the tools at the baseline", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, phpOptions().Tools.Validate(), "Validate")
			})

			t.Run("returns nil for other versions of the packages", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{
					PHPStan:            "phpstan/phpstan@2.3.0",
					StrictRules:        "phpstan/phpstan-strict-rules@2.1.0",
					ExtensionInstaller: "phpstan/extension-installer@1.5.0",
					PHPCSFixer:         "php-cs-fixer/shim@3.96.0",
				}
				assert.NoError(t, tools.Validate(), "Validate")
			})

			tests := []struct {
				name string
				give func(*baseline.Tools)
				want string
			}{
				{
					name: "returns ErrInvalid for another package of phpstan",
					give: func(tools *baseline.Tools) { tools.PHPStan = "phpstan/phpstan-shim@0.12.0" },
					want: `phpstan "phpstan/phpstan-shim@0.12.0", which is not a version of phpstan/phpstan`,
				},
				{
					name: "returns ErrInvalid for another package of phpstan-strict-rules",
					give: func(tools *baseline.Tools) { tools.StrictRules = "phpstan/phpstan-deprecation-rules@2.0.0" },
					want: `phpstan-strict-rules "phpstan/phpstan-deprecation-rules@2.0.0", which is not a version of ` +
						`phpstan/phpstan-strict-rules`,
				},
				{
					name: "returns ErrInvalid for another package of phpstan-extension-installer",
					give: func(tools *baseline.Tools) { tools.ExtensionInstaller = "composer/installers@2.3.0" },
					want: `phpstan-extension-installer "composer/installers@2.3.0", which is not a version of ` +
						`phpstan/extension-installer`,
				},
				{
					name: "returns ErrInvalid for another package of php-cs-fixer",
					give: func(tools *baseline.Tools) { tools.PHPCSFixer = "friendsofphp/php-cs-fixer@3.95.27" },
					want: `php-cs-fixer "friendsofphp/php-cs-fixer@3.95.27", which is not a version of php-cs-fixer/shim`,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					tools := phpOptions().Tools
					tt.give(&tools)
					err := tools.Validate()
					assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
					assert.HasSuffix(t, err.Error(), tt.want, "the error")
				})
			}
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that PHP does not have", func(t *testing.T) {
			t.Parallel()
			o := phpOptions()
			o.Check = option.Check{option.StepFuzz}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command", func(t *testing.T) {
			t.Parallel()
			o := phpOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Command = []string{"composer", "run-script", "generate"}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := phpOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job and the updates of PHP at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, phpOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-php",
					Name:        "PHP",
					Permissions: map[string]string{"contents": "read"},
					Setup: &workflow.Setup{
						Files:    ".php-version",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{{
							Name: "Set up PHP",
							Uses: setupPHP,
							With: map[string]string{"php-version-file": ".php-version"},
						}},
						Timeout: 30,
					},
					Tools: true,
					Steps: []workflow.Step{{Name: "Check PHP", Run: []string{"make check-php"}}},
				}},
				Updates: []workflow.Update{{Ecosystem: "composer", Directories: []string{"/"}}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := phpOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup", func(t *testing.T) {
			t.Parallel()
			o := phpOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			want := slices.Concat(phpOptions().Contribution().Jobs[0].Setup.Steps, o.CI.Steps)
			assert.Equal(t, o.Contribution().Jobs[0].Setup.Steps, want, "the steps of the setup")
		})

		t.Run("installs the PHP of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := phpOptions()
			o.CI.Versions = []string{"8.4", "8.5"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Steps[0].With, map[string]string{"php-version": "${{ matrix.version }}"},
				"the inputs of setup-php")
		})
	})
}

// phpOptions returns new options of the section php at the baseline.
func phpOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
