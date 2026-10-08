// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The modes of the entries of a tree that Restore writes as other than a regular file.
const (
	// modeExecutable is the mode of an executable file.
	modeExecutable = "100755"

	// modeLink is the mode of a symbolic link, whose content is its target.
	modeLink = "120000"
)

// The modes that Restore creates a file and a directory with, as git checks them out, before the
// umask.
const (
	filePerm       fs.FileMode = 0o644
	executablePerm fs.FileMode = 0o755
	dirPerm        fs.FileMode = 0o755
)

// Snapshot returns the tree of the working tree of dir as git would commit it: the files that git
// tracks and the files that git does not track and that no ignore rule excludes, with their content
// in the working tree. It stages the files in an index of its own, so HEAD, the index of the
// repository and every ref keep their values. git removes the objects of the tree when it collects
// garbage after the expiry of unreachable objects, two weeks by default.
//
// It returns an error that wraps [ErrGit] for a dir outside a working tree, and for a ctx that ends
// before git does. It returns the error of the temporary directory of the index.
func Snapshot(ctx context.Context, dir string) (string, error) {
	tmp, err := os.MkdirTemp("", "ergon-snapshot-")
	if err != nil {
		return "", fmt.Errorf("vcs: snapshot %s: %w", dir, err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}
	var tree []byte
	for _, args := range [][]string{{"add", "--all"}, {"write-tree"}} {
		if tree, err = runEnv(ctx, dir, env, Terminal{}, args...); err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(string(tree)), nil
}

// Restore gives each of paths, relative to dir and slash-separated, the content and the kind that
// it has in tree, a tree or a commit, and removes each path that tree does not have, so that the
// working tree has the files of tree at those paths. It creates the missing directories of a
// restored file, and leaves the index of the repository and HEAD unchanged. A restored file keeps
// the mode of the file that it replaces, and a new file takes the mode of its kind in tree under
// the umask.
//
// It returns an error that wraps [ErrGit] for a tree that the repository of dir does not have, for
// a path that is a directory in tree, and for a ctx that ends before git does. It returns the first
// error of a write or a removal, with its path.
func Restore(ctx context.Context, dir, tree string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	out, err := run(ctx, dir, append([]string{"ls-tree", "-z", "--full-tree", tree, "--"}, paths...)...)
	if err != nil {
		return err
	}
	type entry struct{ mode, object string }
	entries := map[string]entry{}
	for line := range bytes.SplitSeq(bytes.TrimSuffix(out, []byte{0}), []byte{0}) {
		meta, path, _ := strings.Cut(string(line), "\t")
		if fields := strings.Fields(meta); len(fields) == 3 {
			entries[path] = entry{mode: fields[0], object: fields[2]}
		}
	}
	for _, path := range paths {
		full := filepath.Join(dir, filepath.FromSlash(path))
		e, ok := entries[path]
		if !ok {
			if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("vcs: restore %s: %w", path, err)
			}
			continue
		}
		data, err := run(ctx, dir, "cat-file", "blob", e.object)
		if err != nil {
			return err
		}
		if err := write(path, full, e.mode, data); err != nil {
			return err
		}
	}
	return nil
}

// write writes data to the file full, at path in the repository, as an entry of a tree of mode,
// after the missing directories of full: a symbolic link to data for the mode of a link, and a
// file with data otherwise, executable for the mode of an executable. It returns the error of the
// write, with path.
func write(path, full, mode string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(full), dirPerm)
	switch {
	case err != nil:
	case mode == modeLink:
		if err = os.Remove(full); err == nil || errors.Is(err, fs.ErrNotExist) {
			err = os.Symlink(string(data), full)
		}
	case mode == modeExecutable:
		err = os.WriteFile(full, data, executablePerm)
	default:
		err = os.WriteFile(full, data, filePerm)
	}
	if err != nil {
		return fmt.Errorf("vcs: restore %s: %w", path, err)
	}
	return nil
}
