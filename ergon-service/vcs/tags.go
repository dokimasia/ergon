// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"context"
	"strings"
)

// Tags returns the commit of each tag of the repository of dir, by the name of the tag without
// refs/tags/: the commit that an annotated tag points to through its tag object, and the commit of
// a lightweight tag. It returns an empty map for a repository without tags, and an error that
// wraps [ErrGit] for a dir outside a working tree and for a ctx that ends before git does.
func Tags(ctx context.Context, dir string) (map[string]string, error) {
	out, err := run(ctx, dir, "for-each-ref", "--format=%(refname:strip=2)%00%(objectname)%00%(*objectname)",
		"refs/tags")
	if err != nil {
		return nil, err
	}
	tags := map[string]string{}
	for line := range strings.Lines(string(out)) {
		name, rest, _ := strings.Cut(strings.TrimSuffix(line, "\n"), "\x00")
		object, peeled, _ := strings.Cut(rest, "\x00")
		tags[name] = object
		if peeled != "" {
			tags[name] = peeled
		}
	}
	return tags, nil
}

// Tag creates the annotated tag name at commit in the repository of dir, with message as its
// annotation, verbatim: git keeps the lines that start with #, such as the headings of a section
// of a changelog. git signs the tag when the configuration of the repository sets tag.gpgSign. It
// returns an error that wraps [ErrGit] for a tag that exists, for a commit that the repository
// does not have, for a signature that fails, and for a ctx that ends before git does.
func Tag(ctx context.Context, dir, name, commit, message string) error {
	_, err := run(ctx, dir, "tag", "--annotate", "--cleanup=verbatim", "--message", message, "--", name, commit)
	return err
}

// Push pushes refs, such as refs/tags/v1.2.0, from the repository of dir to remote in one atomic
// push: the remote updates every ref or none. It returns an error that wraps [ErrGit] for a push
// that the remote refuses or that does not reach it, and for a ctx that ends before git does.
func Push(ctx context.Context, dir, remote string, refs ...string) error {
	_, err := run(ctx, dir, append([]string{"push", "--atomic", "--", remote}, refs...)...)
	return err
}
