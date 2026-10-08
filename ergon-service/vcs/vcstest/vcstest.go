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

// maintenance is the key of the configuration of git that turns the automatic maintenance on or
// off, which [Isolate] sets to false.
const maintenance = "maintenance.auto"

// Isolate removes every variable of the environment whose name starts with GIT_, and then sets
// GIT_CONFIG_GLOBAL to the null device and GIT_CONFIG_NOSYSTEM to 1, so git reads no configuration
// of the user or of the system. It also sets maintenance.auto to false through GIT_CONFIG_COUNT,
// GIT_CONFIG_KEY_0 and GIT_CONFIG_VALUE_0, so no command of git starts the automatic maintenance,
// whose process runs in the background after the command returns. A TestMain calls Isolate before
// m.Run, while no test runs: it changes the environment of the process, which is not safe for
// concurrent use.
func Isolate() {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, prefix) {
			_ = os.Unsetenv(name)
		}
	}
	_ = os.Setenv(prefix+"CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv(prefix+"CONFIG_NOSYSTEM", "1")
	_ = os.Setenv(prefix+"CONFIG_COUNT", "1")
	_ = os.Setenv(prefix+"CONFIG_KEY_0", maintenance)
	_ = os.Setenv(prefix+"CONFIG_VALUE_0", "false")
}

// Repository returns a new working tree of git in a directory of the test, with the files of tree,
// in which git tracks the files of track. The repository commits and tags as Test
// <test@example.invalid> without a signature. It stops the test at a command of git that fails.
func Repository(tb testing.TB, tree files.Tree, track ...string) string {
	tb.Helper()
	dir := files.Workspace(tb, tree)
	Git(tb, dir, "init", "--quiet")
	Git(tb, dir, "config", "user.name", "Test")
	Git(tb, dir, "config", "user.email", "test@example.invalid")
	Git(tb, dir, "config", "commit.gpgSign", "false")
	Git(tb, dir, "config", "tag.gpgSign", "false")
	if len(track) > 0 {
		Git(tb, dir, append([]string{"add", "--"}, track...)...)
	}
	return dir
}

// Git runs git with args in dir, with the hooks of git off, and stops the test when git fails, with
// the output of git in the failure. It returns the standard output and the standard error of git.
func Git(tb testing.TB, dir string, args ...string) string {
	tb.Helper()
	all := append([]string{"-C", dir, "-c", "core.hooksPath=" + os.DevNull}, args...)
	out, err := exec.CommandContext(tb.Context(), "git", all...).CombinedOutput()
	assert.NoError(tb, err, "git "+strings.Join(args, " ")+": "+string(out))
	return string(out)
}

// Commit stages every change of the working tree of dir, a working tree of [Repository], and
// commits it with message, also when nothing changed. It returns the commit, as 40 hexadecimal
// digits, and stops the test when git fails.
func Commit(tb testing.TB, dir, message string) string {
	tb.Helper()
	Git(tb, dir, "add", "--all")
	Git(tb, dir, "commit", "--quiet", "--allow-empty", "--message", message)
	return strings.TrimSpace(Git(tb, dir, "rev-parse", "HEAD"))
}
