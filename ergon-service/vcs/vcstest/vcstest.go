// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcstest

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// prefix starts the name of every variable of the environment that git reads.
const prefix = "GIT_"

// Isolate removes every variable of the environment whose name starts with GIT_, and then sets
// GIT_CONFIG_GLOBAL to the null device and GIT_CONFIG_NOSYSTEM to 1, so git reads no configuration
// of the user or of the system. A TestMain calls it before m.Run, while no test runs: it changes the
// environment of the process, which is not safe for concurrent use.
func Isolate() {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, prefix) {
			_ = os.Unsetenv(name)
		}
	}
	_ = os.Setenv(prefix+"CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv(prefix+"CONFIG_NOSYSTEM", "1")
}

// Repository returns a new working tree of git in a directory of the test, with the files of tree,
// in which git tracks the files of track. It stops the test at a command of git that fails.
func Repository(tb testing.TB, tree files.Tree, track ...string) string {
	tb.Helper()
	dir := files.Workspace(tb, tree)
	Git(tb, dir, "init", "--quiet")
	if len(track) > 0 {
		Git(tb, dir, append([]string{"add", "--"}, track...)...)
	}
	return dir
}

// Git runs git with args in dir, with the hooks of git off, and stops the test when git fails, with
// the output of git in the failure.
func Git(tb testing.TB, dir string, args ...string) {
	tb.Helper()
	all := append([]string{"-C", dir, "-c", "core.hooksPath=" + os.DevNull}, args...)
	out, err := exec.CommandContext(tb.Context(), "git", all...).CombinedOutput()
	assert.NoError(tb, err, "git "+strings.Join(args, " ")+": "+string(out))
}
