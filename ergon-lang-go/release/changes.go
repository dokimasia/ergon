// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// changes are the files of a repository that [Versioner.Apply] changes, with the content of each
// before its first change.
type changes struct {
	// before maps each touched file to its content before the first change, or to nil for a file
	// that did not exist.
	before map[string][]byte

	// root is the root of the repository.
	root string

	// touched are the files that Apply changed or began to change, relative to root and
	// slash-separated, in the order of the first change.
	touched []string
}

// touch records the content of file before its first change. It returns the error of reading
// file, other than the error of a file that does not exist.
func (c *changes) touch(file string) error {
	if _, ok := c.before[file]; ok {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(c.root, filepath.FromSlash(file)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("release: read %s: %w", file, err)
	}
	c.before[file] = data
	c.touched = append(c.touched, file)
	return nil
}

// write touches file and writes data into it. It returns the error of touch or of the write, with
// file.
func (c *changes) write(file string, data []byte) error {
	err := c.touch(file)
	if err == nil {
		err = os.WriteFile(filepath.Join(c.root, filepath.FromSlash(file)), data, filePerm)
	}
	if err != nil {
		return fmt.Errorf("release: write %s: %w", file, err)
	}
	return nil
}

// changed returns the touched files whose content or existence differs from before their first
// change, in the order of touched. A file that it cannot read counts as changed when it existed.
func (c *changes) changed() []string {
	var out []string
	for _, file := range c.touched {
		before := c.before[file]
		now, err := os.ReadFile(filepath.Join(c.root, filepath.FromSlash(file)))
		if (err == nil) != (before != nil) || !bytes.Equal(now, before) {
			out = append(out, file)
		}
	}
	return out
}
