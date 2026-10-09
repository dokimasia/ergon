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
// annotation, verbatim and ending in a newline: git keeps the lines that start with #, such as the
// headings of a section of a changelog, and appends the signature of a signed tag to the
// annotation, which git verify-tag finds only on a line of its own. git signs the tag when the
// configuration of the repository sets tag.gpgSign, and its signing program reads and writes t,
// such as ssh-keygen for the PIN of a security key. It returns an error that wraps [ErrGit] for a
// tag that exists, for a commit that the repository does not have, for a signature that fails, and
// for a ctx that ends before git does.
func Tag(ctx context.Context, dir string, t Terminal, name, commit, message string) error {
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}
	args := []string{"tag", "--annotate", "--cleanup=verbatim", "--message", message, "--", name, commit}
	_, err := runEnv(ctx, dir, nil, t, args...)
	return err
}

// LightTag creates the lightweight tag name at commit in the repository of dir, unsigned whatever
// the configuration of the repository states, such as the tag of a module that a pack builds the
// assets at before the publish creates the same tag on the host. It returns an error that wraps
// [ErrGit] for a tag that exists, for a commit that the repository does not have, and for a ctx that
// ends before git does.
func LightTag(ctx context.Context, dir, name, commit string) error {
	_, err := run(ctx, dir, "tag", "--no-sign", "--", name, commit)
	return err
}

// Push pushes refs, such as refs/tags/v1.2.0, from the repository of dir to remote in one atomic
// push: the remote updates every ref or none. git and its ssh read and write t, such as for the PIN
// and the touch of the key of the remote. It returns an error that wraps [ErrGit] for a push that
// fails to connect to the remote or that the remote refuses, and for a ctx that ends before git
// does.
func Push(ctx context.Context, dir string, t Terminal, remote string, refs ...string) error {
	_, err := runEnv(ctx, dir, nil, t, append([]string{"push", "--atomic", "--", remote}, refs...)...)
	return err
}
