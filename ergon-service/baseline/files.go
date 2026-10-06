// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
)

// ErrUnmanagedLocal is the error for a local file whose path is not a managed file of the
// rendering.
var ErrUnmanagedLocal = errors.New("baseline: local file of a path that is not managed")

// localDir is the directory of the local files: .ergon/local/<path> extends the managed file at
// <path>.
const localDir = ".ergon/local"

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

// applyLocal merges each local file into the managed target of its path, and records the digest
// of the local file. It returns an error that wraps [ErrUnmanagedLocal] for a local file whose
// path is not a managed target, and the error of a local file that does not read or merge.
func (r *Repository) applyLocal(targets []target) error {
	err := fs.WalkDir(r.fsys.FS(), localDir, func(name string, d fs.DirEntry, err error) error {
		if name == localDir && errors.Is(err, fs.ErrNotExist) {
			return fs.SkipAll
		}
		if err != nil {
			return fmt.Errorf("baseline: read %s: %w", name, err)
		}
		if d.IsDir() {
			return nil
		}
		managed := strings.TrimPrefix(name, localDir+"/")
		i := slices.IndexFunc(targets, func(t target) bool {
			return t.path == managed && t.class == language.Managed
		})
		if i < 0 {
			return fmt.Errorf("%w: %s", ErrUnmanagedLocal, name)
		}
		local, err := fs.ReadFile(r.fsys.FS(), name)
		if err != nil {
			return fmt.Errorf("baseline: read %s: %w", name, err)
		}
		content, err := extend(&targets[i], local)
		if err != nil {
			return fmt.Errorf("baseline: merge %s: %w", name, err)
		}
		targets[i].content = content
		targets[i].local = digest(local)
		return nil
	})
	return err
}

// extend returns the content of t with local merged into it: as YAML for a YAML file, with the
// lists of both appended, and appended line by line for any other file.
func extend(t *target, local []byte) ([]byte, error) {
	if isYAML(t.path) {
		return mergeYAML(t.content, local, appendLists)
	}
	content := slices.Clone(t.content)
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		content = append(content, '\n')
	}
	return append(content, local...), nil
}

// configure returns the configured file of t: existing with the keys of t's rendering written
// into it, or the rendering when the file does not exist, as ok reports. It reports whether the
// result differs from existing in its keys and values. Maps are merged key by key, and every
// other value of the rendering replaces the existing one.
func configure(t *target, existing []byte, ok bool) ([]byte, bool, error) {
	if !ok {
		return t.content, true, nil
	}
	// The existing file merged into itself is the file as an encode writes it, so a comparison
	// with the merged file finds a change in keys or values and ignores one of layout.
	normalized, err := mergeYAML(existing, existing, replaceLists)
	if err != nil {
		return nil, false, fmt.Errorf("baseline: configure %s: %w", t.path, err)
	}
	merged, err := mergeYAML(existing, t.content, replaceLists)
	if err != nil {
		return nil, false, fmt.Errorf("baseline: configure %s: rendering: %w", t.path, err)
	}
	return merged, !bytes.Equal(merged, normalized), nil
}

// isYAML reports whether name is a YAML file, by its extension.
func isYAML(name string) bool {
	ext := path.Ext(name)
	return ext == ".yml" || ext == ".yaml"
}
