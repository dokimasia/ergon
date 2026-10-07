// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// ErrGit is the error for a command of git that does not run or that ends with an error, such as
// git ls-files in a directory outside a working tree.
var ErrGit = errors.New("vcs: git failed")

// run runs git with args in dir and returns its standard output. It returns an error that wraps
// [ErrGit] and states args and the standard error of git when git does not run, when it ends with
// an error, and when ctx ends before git does.
func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return runEnv(ctx, dir, nil, args...)
}

// runEnv runs git as [run] does, with the variables of env, each NAME=value, added to the
// environment of the process.
func runEnv(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	all := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: git %s: %w: %s", ErrGit, strings.Join(all, " "), err,
			strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// paths returns the paths of out, the output of git with -z: one path after each NUL byte. It
// returns them sorted, each once, and nil for an empty out.
func paths(out []byte) []string {
	list := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if list[0] == "" {
		return nil
	}
	slices.Sort(list)
	return slices.Compact(list)
}
