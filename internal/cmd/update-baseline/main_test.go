// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// childVar is the variable of the environment that makes the test binary run main in place of its
// tests. A test that runs the binary again sets it, so main runs in a process whose coverage go
// test merges into the profile.
const childVar = "UPDATE_BASELINE_TEST_MAIN"

// fakeArg is the first argument that makes the test binary a program of the cases: it writes its
// working directory to the standard output and its other arguments to the standard error, and
// exits with the status that its second argument states.
const fakeArg = "fake-program"

// usage is the usage of the command, which the package flag writes.
const usage = `Usage of update-baseline:
  -major
    	take a release of a later major version
  -min-age age
    	take a release once it is age old (default 168h0m0s)
  -propose
    	open the pull request of the changes on the branch ergon-update/<base>
`

// aged are the routes of a proxy of the module lintModule at earlier with two newer releases: newer,
// which it published one hour more than seven days before now, and newest, which it published one
// hour less than seven days before now. The cases only read them.
var aged = map[string]string{
	"/" + lintModule + "/@latest":                `{"Version":"v1.2.0","Time":"2026-10-01T13:00:00Z"}`,
	"/" + lintModule + "/@v/list":                earlier + "\n" + newer + "\n" + newest + "\n",
	"/" + lintModule + "/@v/" + newer + ".info":  `{"Version":"v1.1.0","Time":"2026-10-01T11:00:00Z"}`,
	"/" + lintModule + "/@v/" + newest + ".info": `{"Version":"v1.2.0","Time":"2026-10-01T13:00:00Z"}`,
	"/" + formatModule + "/@latest":              `{"Version":"v1.0.0","Time":"2026-09-01T12:00:00Z"}`,
	"/" + formatModule + "/@v/list":              earlier + "\n",
}

func TestMain(m *testing.M) {
	if os.Getenv(childVar) != "" {
		main()
	}
	if len(os.Args) > 2 && os.Args[1] == fakeArg {
		dir, _ := os.Getwd()
		_, _ = fmt.Fprintln(os.Stdout, dir)
		_, _ = fmt.Fprintln(os.Stderr, strings.Join(os.Args[2:], " "))
		status, _ := strconv.Atoi(os.Args[2])
		os.Exit(status)
	}
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestUpdateBaseline(t *testing.T) {
	t.Parallel()

	t.Run("run", func(t *testing.T) {
		t.Parallel()

		usages := []struct {
			name   string
			args   []string
			status int
			stderr string
		}{
			{name: "returns 0 for -h, after the usage", args: []string{"-h"}, status: 0, stderr: usage},
			{
				name:   "returns 2 for a flag that the command does not have, after the usage",
				args:   []string{"-x"},
				status: statusUsage,
				stderr: "flag provided but not defined: -x\n" + usage,
			},
			{
				name:   "returns 2 for an argument, after the usage",
				args:   []string{"extra"},
				status: statusUsage,
				stderr: "update-baseline: the command takes no argument, and has [\"extra\"]\n" + usage,
			},
		}
		for _, tt := range usages {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				fx := newFixture(t)
				var stdout, stderr bytes.Buffer
				status := run(t.Context(), fx.process(&stdout, &stderr, tt.args...))
				assert.Equal(t, status, tt.status, "the exit status")
				expect.Empty(t, stdout.String(), "the standard output")
				expect.Equal(t, stderr.String(), tt.stderr, "the standard error")
				expect.Empty(t, fx.proxy.paths(), "the requests to the proxy")
			})
		}

		t.Run("returns 1 for an update that fails, after its error", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			var stdout, stderr bytes.Buffer
			p := fx.process(&stdout, &stderr)
			p.getwd = func() (string, error) { return "", errWorkingDirectory }
			assert.Equal(t, run(t.Context(), p), statusFailure, "the exit status")
			expect.Equal(t, stderr.String(), "update-baseline: find the working directory: getwd: failed\n",
				"the standard error")
		})

		t.Run("returns 0 for an update that moves no pin", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.proxy.routes["/"+lintModule+"/@v/list"] = earlier + "\n"
			var stdout, stderr bytes.Buffer
			assert.Equal(t, run(t.Context(), fx.process(&stdout, &stderr)), 0, "the exit status")
			expect.Equal(t, stdout.String(), "no pin has a newer release\n", "the standard output")
			expect.Empty(t, stderr.String(), "the standard error")
		})

		updates := []struct {
			name   string
			args   []string
			routes map[string]string
			want   string
		}{
			{
				name:   "takes a release once its registry published it seven days ago by default",
				routes: aged,
				want:   "update tool.tools.lint from v1.0.0 to v1.1.0\n",
			},
			{
				name:   "takes a release once its registry published it the age of -min-age ago",
				args:   []string{"-min-age", "1h"},
				routes: aged,
				want:   "update tool.tools.lint from v1.0.0 to v1.2.0\n",
			},
			{
				name:   "takes the release of a later major version with -major",
				args:   []string{"-major"},
				routes: releases(),
				want:   "update tool.tools.lint from v1.0.0 to v2.0.0\n",
			},
		}
		for _, tt := range updates {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				fx := newFixture(t)
				fx.proxy.routes = tt.routes
				var stdout, stderr bytes.Buffer
				assert.Equal(t, run(t.Context(), fx.process(&stdout, &stderr, tt.args...)), 0,
					"the exit status: "+stderr.String())
				expect.HasPrefix(t, stdout.String(), tt.want, "the standard output")
				expect.Empty(t, fx.hub.calls(), "the requests to GitHub")
			})
		}

		t.Run("proposes the changes with -propose", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			var stdout, stderr bytes.Buffer
			assert.Equal(t, run(t.Context(), fx.process(&stdout, &stderr, "-propose")), 0,
				"the exit status: "+stderr.String())
			expect.HasSuffix(t, stdout.String(), "pull request 7\n", "the standard output")
			expect.Equal(t, fx.hub.opened(), []string{proposed}, "the bodies of the pull requests")
		})
	})
}

func TestUpdateBaselineProcess(t *testing.T) {
	binary, err := os.Executable()
	assert.NoError(t, err, "the path of the test binary")

	t.Run("main", func(t *testing.T) {
		t.Run("exits 0 for -h, after the usage", func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), binary, "-h")
			cmd.Env = append(os.Environ(), childVar+"=1")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			assert.NoError(t, cmd.Run(), "update-baseline -h")
			expect.Empty(t, stdout.String(), "the standard output")
			expect.Equal(t, stderr.String(), usage, "the standard error")
		})
	})

	t.Run("execute", func(t *testing.T) {
		t.Run("runs the program with the arguments in the directory", func(t *testing.T) {
			dir := t.TempDir()
			var stdout, stderr bytes.Buffer
			assert.NoError(t, execute(t.Context(), dir, &stdout, &stderr, binary, fakeArg, "0", "second"),
				"execute")
			expect.Equal(t, stdout.String(), dir+"\n", "the standard output, the working directory of the program")
			expect.Equal(t, stderr.String(), "0 second\n", "the standard error, the arguments of the program")
		})

		t.Run("returns the error of an exit status other than 0 with the command line", func(t *testing.T) {
			err := execute(t.Context(), t.TempDir(), io.Discard, io.Discard, binary, fakeArg, "3")
			exit := assert.ErrorAs[*exec.ExitError](t, err, "the error of execute")
			expect.Equal(t, exit.ExitCode(), 3, "the exit status of the program")
			expect.Equal(t, err.Error(), binary+" "+fakeArg+" 3: exit status 3", "the text of the error")
		})

		t.Run("returns the error of a program that does not start", func(t *testing.T) {
			missing := filepath.Join(t.TempDir(), "missing")
			want := exec.CommandContext(t.Context(), missing, "first").Run()
			err := execute(t.Context(), t.TempDir(), io.Discard, io.Discard, missing, "first")
			assert.Equal(t, errors.Unwrap(err), want, "the error that execute wraps, the error of os/exec")
			expect.HasPrefix(t, err.Error(), missing+" first: ", "the text of the error")
		})
	})
}
