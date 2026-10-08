// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"context"
	"strings"
)

// Head returns the commit of HEAD of the working tree of dir, as 40 hexadecimal digits. It returns
// an error that wraps [ErrGit] for a dir outside a working tree, for a repository without a commit,
// and for a ctx that ends before git does.
func Head(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HeadTree returns the commit of HEAD of the working tree of dir and the tree of that commit, each
// as 40 hexadecimal digits. It returns an error that wraps [ErrGit] for a dir outside a working
// tree, for a repository without a commit, and for a ctx that ends before git does.
func HeadTree(ctx context.Context, dir string) (string, string, error) {
	out, err := run(ctx, dir, "rev-parse", "HEAD^{commit}", "HEAD^{tree}")
	if err != nil {
		return "", "", err
	}
	commit, tree, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return commit, tree, nil
}

// AddedBy returns the newest commit of the history of HEAD that added or copied the file at path,
// relative to dir, following the file across renames, as changesets finds the commit of a
// changeset. It returns the empty string for a file that no commit added, such as a file that only
// the working tree has. It returns an error that wraps [ErrGit] for a dir outside a working tree
// and for a ctx that ends before git does.
func AddedBy(ctx context.Context, dir, path string) (string, error) {
	out, err := run(ctx, dir, "log", "--diff-filter=AC", "--follow", "--max-count=1", "--format=%H", "--", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
