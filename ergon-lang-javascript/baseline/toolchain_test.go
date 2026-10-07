// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// toolchainName pins the name of the producer of the js toolchain, which is the name of its
// section.
const toolchainName = "js"

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

			t.Run("names the schema of the Biome of the section js", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Tools.Biome = "@biomejs/biome@2.6.0"
				u := render.Unit{Name: baseline.ToolchainName, Producer: baseline.Toolchain{}, Options: o}
				config := rendered(t, u, "biome.json")
				assert.Contains(t, config, `"$schema": "https://biomejs.dev/schemas/2.6.0/schema.json",`, "biome.json")
			})

			t.Run("scans the packages at the severity of the section js", func(t *testing.T) {
				t.Parallel()
				o := jsOptions()
				o.Audit.Severity = option.SeverityHigh
				u := render.Unit{Name: baseline.ToolchainName, Producer: baseline.Toolchain{}, Options: o}
				makefile := rendered(t, u, "Makefile")
				assert.Contains(t, makefile, "\nJS_AUDIT_SEVERITY ?= high\n", "the Makefile")
			})
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
