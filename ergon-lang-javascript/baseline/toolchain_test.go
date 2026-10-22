// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// toolchainName pins the name of the producer of the js toolchain, which is the name of its
// section.
const toolchainName = "js"

// prettierIgnore is the .prettierignore of the js toolchain alone for the toolkit vite-plus. It lists
// .ergon/, .ergon.yaml and the managed files of the rendering.
const prettierIgnore = "# Managed by ergon init. Add repository settings to .ergon/local/.prettierignore " +
	"and run ergon init sync.\n" +
	"#\n" +
	"# The files that ergon init writes, which vp fmt leaves as ergon renders them.\n" +
	"/.ergon/\n" +
	"/.ergon.yaml\n" +
	"/.gitattributes\n" +
	"/.gitignore\n" +
	"/.prettierignore\n" +
	"/Makefile\n"

func TestToolchain(t *testing.T) {
	t.Parallel()

	t.Run("ToolchainName", func(t *testing.T) {
		t.Parallel()

		t.Run("is js", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.ToolchainName, toolchainName, "ToolchainName")
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the schema of the Biome of the section js", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Tools.Biome = "@biomejs/biome@2.6.0"
				config := rendered(t, "biome.json", o)
				assert.Contains(t, config, `"$schema": "https://biomejs.dev/schemas/2.6.0/schema.json",`, "biome.json")
			})

			t.Run("scans the packages at the severity of the section js", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Audit.Severity = option.SeverityHigh
				makefile := rendered(t, "Makefile", o)
				assert.Contains(t, makefile, "\nJS_AUDIT_SEVERITY ?= high\n", "the Makefile")
			})

			t.Run("runs the targets with npm at the baseline", func(t *testing.T) {
				t.Parallel()
				makefile := rendered(t, "Makefile", jsOptions())
				expect.That(t, makefile).
					Contains("\nJS_TEST := npm test --\n", "the command of the tests").
					Contains("\naudit-js: ## Scan package-lock.json for known vulnerabilities\n"+
						"\tnpm audit --audit-level=$(JS_AUDIT_SEVERITY) --include=dev\n", "the scan of npm")
			})

			t.Run("runs the targets with pnpm for the package manager pnpm", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.PackageManager = option.PackageManagerPNPM
				makefile := rendered(t, "Makefile", o)
				expect.That(t, makefile).
					Contains("\nJS_TEST := pnpm test --\n", "the command of the tests").
					Contains("\naudit-js: ## Scan pnpm-lock.yaml for known vulnerabilities\n"+
						"\tpnpm audit --audit-level=$(JS_AUDIT_SEVERITY)\n", "the scan of pnpm")
			})

			t.Run("runs vp of the packages with npm exec for the toolkit vite-plus", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Toolkit = baseline.ToolkitVitePlus
				makefile := rendered(t, "Makefile", o)
				expect.That(t, makefile).
					Contains("\nVP := npm exec -- vp\n", "the vp of the packages").
					Contains("\nJS_TEST := $(VP) test --run\n", "the command of the tests").
					Contains("\n\t$(VP) fmt\n", "the formatter").
					Contains("\n\t$(VP) check\n", "the check").
					NotContains("BIOME", "the Biome of the toolkit biome").
					NotContains("JS_PATHS", "the paths of Biome")
			})

			t.Run("runs vp of the packages with pnpm exec for vite-plus with pnpm", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.PackageManager = option.PackageManagerPNPM
				o.Toolkit = baseline.ToolkitVitePlus
				makefile := rendered(t, "Makefile", o)
				assert.Contains(t, makefile, "\nVP := pnpm exec vp\n", "the vp of the packages")
			})

			t.Run("does not render .prettierignore for the toolkit biome", func(t *testing.T) {
				t.Parallel()
				assert.NotContains(t, paths(t, jsOptions()), ".prettierignore", "the paths of the files")
			})

			t.Run("does not render biome.json for the toolkit vite-plus", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Toolkit = baseline.ToolkitVitePlus
				assert.NotContains(t, paths(t, o), "biome.json", "the paths of the files")
			})

			t.Run("lists .ergon, .ergon.yaml and the managed files in .prettierignore for vite-plus",
				func(t *testing.T) {
					t.Parallel()
					o := jsOptions()
					o.Toolkit = baseline.ToolkitVitePlus
					assert.Equal(t, rendered(t, ".prettierignore", o), prettierIgnore, ".prettierignore")
				})

			t.Run("excludes the globs of exclude in files.includes of biome.json", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Exclude = baseline.Exclude{"vendor", "src/**/*.gen.ts"}
				assert.Contains(t, rendered(t, "biome.json", o), "\n  },\n  \"files\": {\n"+
					"    \"includes\": [\"**\", \"!vendor\", \"!src/**/*.gen.ts\"]\n  },\n  \"formatter\": {\n",
					"biome.json")
			})

			t.Run("keeps files.includes on one line of 88 columns", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Exclude = baseline.Exclude{strings.Repeat("a", 61)}
				line := "    \"includes\": [\"**\", \"!" + strings.Repeat("a", 61) + "\"]\n"
				assert.Length(t, strings.TrimSuffix(line, "\n"), 88, "the line of files.includes")
				assert.Contains(t, rendered(t, "biome.json", o), line, "biome.json")
			})

			t.Run("puts each item of files.includes on a line of its own past 88 columns", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Exclude = baseline.Exclude{strings.Repeat("a", 62), "vendor"}
				assert.Contains(t, rendered(t, "biome.json", o), "    \"includes\": [\n      \"**\",\n"+
					"      \"!"+strings.Repeat("a", 62)+"\",\n      \"!vendor\"\n    ]\n", "biome.json")
			})

			t.Run("does not render the key files in biome.json without exclusions", func(t *testing.T) {
				t.Parallel()
				assert.NotContains(t, rendered(t, "biome.json", jsOptions()), "\"files\"", "biome.json")
			})

			t.Run("lists the globs of exclude in .prettierignore for vite-plus", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Toolkit = baseline.ToolkitVitePlus
				o.Exclude = baseline.Exclude{"vendor", "**/dist"}
				assert.HasSuffix(t, rendered(t, ".prettierignore", o), "\n/Makefile\n\n"+
					"# vp fmt skips the paths of the key exclude of the section js.\n/vendor\n/**/dist\n",
					".prettierignore")
			})

			t.Run("lists the globs of exclude in .eslintignore for vite-plus", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Toolkit = baseline.ToolkitVitePlus
				o.Exclude = baseline.Exclude{"vendor", "**/dist"}
				assert.Equal(t, rendered(t, ".eslintignore", o), "# Managed by ergon init. Add repository settings "+
					"to .ergon/local/.eslintignore and run ergon init sync.\n#\n"+
					"# The lint of vp check skips the paths of the key exclude of the section js.\n/vendor\n/**/dist\n",
					".eslintignore")
			})

			tests := []struct {
				name        string
				giveToolkit baseline.Toolkit
				giveExclude baseline.Exclude
			}{
				{
					name:        "does not render .eslintignore for vite-plus without exclusions",
					giveToolkit: baseline.ToolkitVitePlus,
				},
				{
					name:        "does not render .eslintignore for biome",
					giveToolkit: baseline.ToolkitBiome,
					giveExclude: baseline.Exclude{"vendor"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					o := jsOptions()
					o.Toolkit = tt.giveToolkit
					o.Exclude = tt.giveExclude
					assert.NotContains(t, paths(t, o), ".eslintignore", "the paths of the files")
				})
			}
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first := jsOptions()
				second := jsOptions()
				first.Paths[0] = "src"
				assert.Equal(t, second.Paths[0], ".", "the path of the second value")
			})
		})

		t.Run("Data", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the choices of the package manager and the toolkit of the options", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.PackageManager = option.PackageManagerPNPM
				o.Toolkit = baseline.ToolkitVitePlus
				got, err := baseline.Toolchain{}.Data(nil, o, &workflow.Contribution{})
				assert.NoError(t, err, "Data")
				assert.Equal(t, got, any(baseline.Choices{LineWidth: 88, PNPM: true, VitePlus: true}), "the choices")
			})

			t.Run("returns the choices of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				got, err := baseline.Toolchain{}.Data(nil, nil, &workflow.Contribution{})
				assert.NoError(t, err, "Data")
				assert.Equal(t, got, any(baseline.Choices{LineWidth: 88}), "the choices")
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the contribution of the options", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.CI.Timeout = 45
				assert.Equal(t, baseline.Toolchain{}.Contribution(o), o.Contribution(), "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Toolchain{}.Contribution(nil), jsOptions().Contribution(), "the contribution")
			})
		})
	})
}

// jsOptions returns new options of the section js at the baseline.
func jsOptions() *baseline.ToolchainOptions {
	o, _ := baseline.Toolchain{}.Options().(*baseline.ToolchainOptions)
	return o
}

// paths returns the paths of the files that the js toolchain with the options o renders for the
// answers of the cases, and stops the test at an error of Render.
func paths(t *testing.T, o *baseline.ToolchainOptions) []string {
	t.Helper()
	u := render.Unit{Name: baseline.ToolchainName, Producer: baseline.Toolchain{}, Options: o}
	files, err := render.Render([]render.Unit{u}, baselinetest.Answers(javascript.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Path)
	}
	return names
}
