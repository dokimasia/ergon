// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// childVar is the variable of the environment that makes the test binary run main in place of its
// tests. A test that runs the binary again sets it, so main runs in a process whose coverage go
// test merges into the profile.
const childVar = "ERGON_GO_VET_MAIN"

// The findings of the module of the cases, each after the path of its file in the module, which
// ergon-go-vet prints with the separator of the system.
var (
	demoPrefix = filepath.FromSlash("demo/demo.go") +
		`:6:14: the text "bad" of errors.New does not start with "demo:", the name of its package`
	demoSkip = filepath.FromSlash("demo/demo_test.go") +
		":6:2: the skip expired on 2020-01-01, so fix the test and remove the skip"
	legacy = filepath.FromSlash("legacy/legacy.go") +
		`:6:14: the text "old" of errors.New does not start with "legacy:"`
)

// module is the module of the cases: the package demo breaks both rules, and the package legacy
// breaks the rule of errorprefix.
var module = files.Tree{
	"go.mod": files.Text("module example.com/demo\n\ngo 1.27\n"),
	"demo/demo.go": files.Text(
		"package demo\n\nimport \"errors\"\n\n// ErrBad has no prefix.\nvar ErrBad = errors.New(\"bad\")\n",
	),
	"demo/demo_test.go": files.Text(
		"package demo\n\nimport \"testing\"\n\nfunc TestOld(t *testing.T) {\n\tt.Skip(\"expires 2020-01-01\")\n}\n",
	),
	"legacy/legacy.go": files.Text(
		"package legacy\n\nimport \"errors\"\n\n// ErrOld has no prefix.\nvar ErrOld = errors.New(\"old\")\n",
	),
}

func TestMain(m *testing.M) {
	if os.Getenv(childVar) != "" {
		main()
	}
	os.Exit(m.Run())
}

// TestErgonGoVetProcess runs the test binary as ergon-go-vet, one process at a time, because two
// processes that exit in the same nanosecond write one file of coverage.
func TestErgonGoVetProcess(t *testing.T) {
	t.Run("main", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			env     []string
			status  int
			reports []string
			omits   []string
		}{
			{
				name:    "exits 3 with the findings of both analyzers",
				args:    []string{"./..."},
				status:  3,
				reports: []string{demoPrefix, demoSkip, legacy},
			},
			{
				name:    "skips the packages of each pattern of -exclude",
				args:    []string{"-exclude=./legacy/...", "./..."},
				status:  3,
				reports: []string{demoPrefix, demoSkip},
				omits:   []string{legacy},
			},
			{
				name:   "exits 0 for packages without a finding",
				args:   []string{"-exclude=./legacy/...", "-exclude=./demo", "./..."},
				status: 0,
				omits:  []string{demoPrefix, demoSkip, legacy},
			},
			{
				name:    "exits 2 for a pattern of -exclude that go list does not resolve",
				args:    []string{"-exclude=./missing", "./..."},
				status:  2,
				reports: []string{`invalid value "./missing" for flag -exclude: `, "-exclude pattern"},
			},
			{
				name:    "exits 2 for a pattern of -exclude without the go command",
				args:    []string{"-exclude=./legacy/...", "./..."},
				env:     []string{"PATH="},
				status:  2,
				reports: []string{`invalid value "./legacy/..." for flag -exclude: `},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				status, stderr := vet(t, files.Workspace(t, module), tt.env, tt.args...)
				assert.Equal(t, status, tt.status, "the exit status: "+stderr)
				for _, line := range tt.reports {
					assert.Contains(t, stderr, line, "the standard error")
				}
				for _, line := range tt.omits {
					assert.NotContains(t, stderr, line, "the standard error")
				}
			})
		}
	})
}

// vet runs the test binary as ergon-go-vet with args in the module dir, outside any workspace of
// Go, with the variables of env after those of the test, and returns its exit status and its
// standard error.
func vet(t *testing.T, dir string, env []string, args ...string) (int, string) {
	t.Helper()
	binary, err := os.Executable()
	assert.NoError(t, err, "the path of the test binary")
	cmd := exec.CommandContext(t.Context(), binary, args...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), childVar+"=1", "GOWORK=off"), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err = cmd.Run()
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return exit.ExitCode(), stderr.String()
	}
	assert.NoError(t, err, "the run of ergon-go-vet")
	return 0, stderr.String()
}
