// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"os/exec"
	"testing"

	"go.dokimi.dev/assert"
)

// childVar is the variable of the environment that makes the test binary run main in place of
// its tests. A test that runs the binary again sets it, so main runs in a process whose coverage
// go test merges into the profile.
const childVar = "ERGON_TEST_MAIN"

func TestMain(m *testing.M) {
	if os.Getenv(childVar) != "" {
		main()
	}
	os.Exit(m.Run())
}

func TestErgon(t *testing.T) {
	t.Parallel()

	t.Run("main", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the version of a build without the flags of the linker", func(t *testing.T) {
			t.Parallel()
			binary, err := os.Executable()
			assert.NoError(t, err, "the path of the test binary")
			cmd := exec.CommandContext(t.Context(), binary, "--version")
			cmd.Env = append(os.Environ(), childVar+"=1")
			out, err := cmd.Output()
			assert.NoError(t, err, "ergon --version")
			assert.Equal(t, string(out), "ergon dev\n", "the standard output of ergon --version")
		})
	})
}
