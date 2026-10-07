// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// ErrGit is the error for a command of git that does not run or that ends with an error, such as
// git ls-files in a directory outside a working tree.
var ErrGit = errors.New("vcs: git failed")

// Files returns the files of the working tree of dir that git tracks, and the files that it does
// not track and that no ignore rule excludes, as git ls-files --cached --others --exclude-standard
// lists them. Each path is relative to dir and slash-separated, and the list is sorted and names
// each path once. A tracked file that the working tree has deleted is in the list.
//
// It returns an error that wraps [ErrGit] for a dir that is not in a working tree of git, for a
// machine without git, and for a ctx that ends before git does.
func Files(ctx context.Context, dir string) ([]string, error) {
	args := []string{"-C", dir, "ls-files", "--cached", "--others", "--exclude-standard", "-z"}
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: git %s: %w: %s", ErrGit, strings.Join(args, " "), err,
			strings.TrimSpace(stderr.String()))
	}
	files := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if files[0] == "" {
		return nil, nil
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}
