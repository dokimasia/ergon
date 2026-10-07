// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"

	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/baseline/overlay"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// ErrUnmanagedLocal is the error for a local file whose path is not a managed file of the
// rendering.
var ErrUnmanagedLocal = errors.New("baseline: local file of a path that is not managed")

// The modes of the files and the directories that a command creates, before the umask.
const (
	filePerm fs.FileMode = 0o644
	dirPerm  fs.FileMode = 0o755
)

// FS is the file system of a repository. An [os.Root] of the repository's directory implements
// it, and confines every path to that directory.
type FS interface {
	// ReadFile returns the content of the file name.
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to the file name, which it creates with perm before the umask.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// MkdirAll creates the directory name and every missing parent.
	MkdirAll(name string, perm fs.FileMode) error

	// Rename renames the file oldname to newname, and replaces a file at newname.
	Rename(oldname, newname string) error

	// Remove removes the file name.
	Remove(name string) error

	// FS returns the file system as an [fs.FS], for walking the local files.
	FS() fs.FS
}

var _ FS = (*os.Root)(nil)

// read returns the content of the file name, and reports whether it exists. It returns an error
// for a file that exists and cannot be read, such as a directory.
func (r *Repository) read(name string) ([]byte, bool, error) {
	data, err := r.fsys.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("baseline: read %s: %w", name, err)
	}
	return data, true, nil
}

// write writes data to the file name through a temporary file in the same directory, which it
// renames to name, so a reader sees the old file or the new one. It creates the directories of
// name, and removes the temporary file when the rename fails.
func (r *Repository) write(name string, data []byte) error {
	dir := path.Dir(name)
	if err := r.fsys.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("baseline: create %s: %w", dir, err)
	}
	temporary := path.Join(dir, "."+path.Base(name)+"."+rand.Text())
	if err := r.fsys.WriteFile(temporary, data, filePerm); err != nil {
		return fmt.Errorf("baseline: write %s: %w", name, err)
	}
	if err := r.fsys.Rename(temporary, name); err != nil {
		return errors.Join(fmt.Errorf("baseline: write %s: %w", name, err), r.fsys.Remove(temporary))
	}
	return nil
}

// remove removes the file name.
func (r *Repository) remove(name string) error {
	if err := r.fsys.Remove(name); err != nil {
		return fmt.Errorf("baseline: remove %s: %w", name, err)
	}
	return nil
}

// applyLocal merges each local file into the managed target of its path, as [overlay.Apply]
// states, and records the digest of the local file. It returns an error that wraps
// [ErrUnmanagedLocal] for a local file whose path is not a managed target, and the error of a
// local file that does not read or merge.
func (r *Repository) applyLocal(targets []target) error {
	locals, err := overlay.Read(r.fsys.FS())
	if err != nil {
		return err
	}
	for _, l := range locals {
		i := slices.IndexFunc(targets, func(t target) bool { return t.Path == l.Path && t.Class == render.Managed })
		if i < 0 {
			return fmt.Errorf("%w: %s/%s", ErrUnmanagedLocal, overlay.Dir, l.Path)
		}
		content, err := overlay.Apply(l.Path, targets[i].Content, l.Content)
		if err != nil {
			return err
		}
		targets[i].Content = content
		targets[i].local = lock.Digest(l.Content)
	}
	return nil
}
