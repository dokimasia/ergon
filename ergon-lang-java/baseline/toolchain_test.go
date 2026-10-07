// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

// toolchainName pins the name of the producer of the jvm toolchain, which is the name of its
// section.
const toolchainName = "jvm"

func TestToolchain(t *testing.T) {
	t.Parallel()

	t.Run("ToolchainName", func(t *testing.T) {
		t.Parallel()

		t.Run("is jvm", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.ToolchainName, toolchainName, "ToolchainName")
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first := jvmOptions()
				second := jvmOptions()
				first.Tools.OSVScanner.SHA256[option.LinuxAMD64] = "0"
				assert.Equal(
					t,
					second.Tools.OSVScanner.SHA256[option.LinuxAMD64],
					"ca69b3d3cd08f889a49dc0a383122f71cc528b83803671df5fd874d97485b108",
					"the digest of the second value",
				)
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the contribution of the options", func(t *testing.T) {
				t.Parallel()
				o := jvmOptions()
				o.CI.Timeout = 45
				assert.Equal(t, baseline.Toolchain{}.Contribution(o), o.Contribution(), "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Toolchain{}.Contribution(nil), jvmOptions().Contribution(), "the contribution")
			})
		})
	})
}

// jvmOptions returns new options of the section jvm at the baseline.
func jvmOptions() *baseline.ToolchainOptions {
	o, _ := baseline.Toolchain{}.Options().(*baseline.ToolchainOptions)
	return o
}
