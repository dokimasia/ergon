// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"context"
	"errors"
	"slices"
)

// Files returns the files of the working tree of dir that git tracks, and the files that it does
// not track and that no ignore rule excludes, as git ls-files --cached --others --exclude-standard
// lists them. Each path is relative to dir and slash-separated, and the list is sorted and names
// each path once. A tracked file that the working tree has deleted is in the list.
//
// It returns an error that wraps [ErrGit] for a dir that is not in a working tree of git, for a
// machine without git, and for a ctx that ends before git does.
func Files(ctx context.Context, dir string) ([]string, error) {
	out, err := run(ctx, dir, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	return paths(out), nil
}

// Changed returns the files of dir that differ from the merge base of base and HEAD: the tracked
// files that the commits since the merge base or the working tree add, change, remove or rename,
// and the files that git does not track and that no ignore rule excludes. A rename lists its old
// and its new path. Each path is relative to dir and slash-separated, and the list is sorted and
// names each path once.
//
// It returns an error that wraps [ErrGit] for a base that names no commit, for a base without a
// common ancestor with HEAD, for a dir that is not in a working tree of git, and for a ctx that
// ends before git does.
func Changed(ctx context.Context, dir, base string) ([]string, error) {
	diff, diffErr := run(ctx, dir, "diff", "--merge-base", "--name-only", "--no-renames", "-z", base, "--")
	untracked, untrackedErr := run(ctx, dir, "ls-files", "--others", "--exclude-standard", "-z")
	if err := errors.Join(diffErr, untrackedErr); err != nil {
		return nil, err
	}
	changed := slices.Concat(paths(diff), paths(untracked))
	slices.Sort(changed)
	return slices.Compact(changed), nil
}
