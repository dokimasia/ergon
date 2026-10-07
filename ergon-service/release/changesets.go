// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/changeset"
)

// slugWords is the number of words of a summary that the ID of a new changeset keeps.
const slugWords = 5

// changesetDirPerm is the mode of a new directory .changeset, before the umask.
const changesetDirPerm fs.FileMode = 0o755

// notChangesets are the Markdown files of the directory of the changesets that changesets reads as
// no changeset, besides README.md in any case: the instructions for agents.
var notChangesets = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}

// ReadChangesets returns the changesets in the directory .changeset of the repository at root, in
// the order of their file names. It reads every file that ends in .md except the files that
// changesets skips: README.md in any case, AGENTS.md, CLAUDE.md, GEMINI.md, and a file whose name
// starts with a dot.
//
// It returns the error of reading the directory or a file, which wraps [fs.ErrNotExist] for a
// repository without .changeset, and the error of [changeset.Parse], which names the file and the
// line.
func ReadChangesets(root string) ([]changeset.Changeset, error) {
	dir := filepath.Join(root, changeset.Dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("release: read the changesets: %w", err)
	}
	sets := []changeset.Changeset{}
	for _, e := range entries {
		name := e.Name()
		skipped := e.IsDir() || !strings.HasSuffix(name, changeset.Ext) || strings.HasPrefix(name, ".") ||
			strings.EqualFold(name, changeset.Readme) || slices.Contains(notChangesets, name)
		if skipped {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("release: read the changesets: %w", err)
		}
		s, err := changeset.Parse(strings.TrimSuffix(name, changeset.Ext), data)
		if err != nil {
			return nil, fmt.Errorf("release: %w", err)
		}
		sets = append(sets, s)
	}
	return sets, nil
}

// AddChangeset writes c into the directory .changeset of the repository at root, as the file of its
// ID, in the format of [changeset.Format], and returns the path of the file relative to root and
// slash-separated. It creates a missing directory .changeset, and writes through an [os.Root] of
// root, so an ID that leaves the repository fails. It returns an error that wraps [fs.ErrExist] for
// an ID that a file has, the error of opening root, and the error of the write.
func AddChangeset(root string, c *changeset.Changeset) (string, error) {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("release: open the repository: %w", err)
	}
	defer func() { _ = dir.Close() }()
	if err := dir.MkdirAll(changeset.Dir, changesetDirPerm); err != nil {
		return "", fmt.Errorf("release: create %s: %w", changeset.Dir, err)
	}
	file := changesetPath(c.ID)
	if _, err := dir.Lstat(filepath.FromSlash(file)); err == nil {
		return "", fmt.Errorf("release: create %s: %w", file, fs.ErrExist)
	}
	if err := dir.WriteFile(filepath.FromSlash(file), changeset.Format(c), filePerm); err != nil {
		return "", fmt.Errorf("release: write %s: %w", file, err)
	}
	return file, nil
}

// ChangesetID returns the ID of a new changeset with summary: the first words of its first line in
// lowercase, joined by hyphens, and four hexadecimal digits from random, such as
// add-the-unit-type-3f9a. A summary without a letter or a digit gives the ID of the digits alone. It
// returns the error of reading random.
func ChangesetID(summary string, random io.Reader) (string, error) {
	var suffix [2]byte
	if _, err := io.ReadFull(random, suffix[:]); err != nil {
		return "", fmt.Errorf("release: read the random digits of a changeset: %w", err)
	}
	line, _, _ := strings.Cut(strings.ToLower(summary), "\n")
	words := strings.FieldsFunc(line, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') })
	words = append(words[:min(len(words), slugWords)], hex.EncodeToString(suffix[:]))
	return strings.Join(words, "-"), nil
}
