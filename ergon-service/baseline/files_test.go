// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
)

// errFault is the error of an operation that a faulty file system fails.
var errFault = errors.New("fault: injected")

// The operations that a faulty file system can fail.
const (
	opRead   = "read"
	opWrite  = "write"
	opMkdir  = "mkdir"
	opRename = "rename"
	opRemove = "remove"
	opOpen   = "open"
)

// faulty is the os.Root of a directory that fails one operation on one path. A failed write is
// the write of the temporary file of the path, and a failed read is a read after the first reads
// of the path that after counts.
type faulty struct {
	*os.Root

	// op is the operation that fails.
	op string

	// name is the path whose operation fails.
	name string

	// after is the number of reads of name that succeed before a read fails.
	after int

	// reads counts the reads of name.
	reads int
}

// ReadFile fails for name after f.after reads of it, and otherwise returns the read of the
// directory, with its error wrapped.
func (f *faulty) ReadFile(name string) ([]byte, error) {
	if f.op == opRead && name == f.name {
		f.reads++
		if f.reads > f.after {
			return nil, errFault
		}
	}
	data, err := f.Root.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("faulty: %w", err)
	}
	return data, nil
}

// WriteFile fails for the temporary file of name, and otherwise writes to the directory, with
// its error wrapped.
func (f *faulty) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if f.op == opWrite && path.Dir(name) == path.Dir(f.name) &&
		strings.HasPrefix(path.Base(name), "."+path.Base(f.name)+".") {

		return errFault
	}
	if err := f.Root.WriteFile(name, data, perm); err != nil {
		return fmt.Errorf("faulty: %w", err)
	}
	return nil
}

// MkdirAll fails for name, and otherwise creates the directories, with its error wrapped.
func (f *faulty) MkdirAll(name string, perm fs.FileMode) error {
	if f.op == opMkdir && name == f.name {
		return errFault
	}
	if err := f.Root.MkdirAll(name, perm); err != nil {
		return fmt.Errorf("faulty: %w", err)
	}
	return nil
}

// Rename fails for a rename to name, and otherwise renames, with its error wrapped.
func (f *faulty) Rename(oldname, newname string) error {
	if f.op == opRename && newname == f.name {
		return errFault
	}
	if err := f.Root.Rename(oldname, newname); err != nil {
		return fmt.Errorf("faulty: %w", err)
	}
	return nil
}

// Remove fails for name, and otherwise removes, with its error wrapped.
func (f *faulty) Remove(name string) error {
	if f.op == opRemove && name == f.name {
		return errFault
	}
	if err := f.Root.Remove(name); err != nil {
		return fmt.Errorf("faulty: %w", err)
	}
	return nil
}

// FS returns the file system of the directory, whose Open fails for name.
func (f *faulty) FS() fs.FS {
	if f.op == opOpen {
		return failingFS{FS: f.Root.FS(), name: f.name}
	}
	return f.Root.FS()
}

// failingFS is a file system whose Open fails for name.
type failingFS struct {
	fs.FS

	// name is the path whose Open fails.
	name string
}

// Open fails for f.name, and opens any other file of the file system, with its error wrapped.
func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.name {
		return nil, errFault
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failing: %w", err)
	}
	return file, nil
}

func TestFiles(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("New", func(t *testing.T) {
			t.Parallel()

			t.Run("returns an error for a managed path that is a directory", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				assert.NoError(t, root.Mkdir(license, 0o755), "Mkdir of LICENSE")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
				assert.Contains(t, err.Error(), "baseline: read LICENSE", "the error")
			})

			t.Run("returns an error for a lock that does not read", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				assert.NoError(t, root.MkdirAll(lockPath, 0o755), "MkdirAll of the lock")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
			})

			t.Run("returns an error when a directory cannot be created", func(t *testing.T) {
				t.Parallel()
				_, err := repository(t, fault(t, opMkdir, "alpha")).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
			})

			t.Run("returns an error when a file cannot be written", func(t *testing.T) {
				t.Parallel()
				changes, err := repository(t, fault(t, opWrite, license)).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
				assert.Equal(t, paths(changes), []string{config, workflow, ignore}, "the files that New wrote first")
			})

			t.Run("removes the temporary file when a rename fails", func(t *testing.T) {
				t.Parallel()
				fsys := fault(t, opRename, license)
				_, err := repository(t, fsys).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
				entries, err := fs.ReadDir(fsys.Root.FS(), ".")
				assert.NoError(t, err, "ReadDir of the directory")
				for _, e := range entries {
					assert.False(t, strings.HasPrefix(e.Name(), ".LICENSE."), "a temporary file "+e.Name())
				}
			})

			t.Run("returns an error when the lock cannot be written", func(t *testing.T) {
				t.Parallel()
				_, err := repository(t, fault(t, opWrite, lockPath)).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
			})

			t.Run("appends a local file to a managed file that is not YAML", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+ignore, "local/\n")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, ignore), "# common\nalpha/\nlocal/\n", "the extended .gitignore")
				assert.Contains(t, content(t, root, lockPath), `"local": "`+sum("local/\n")+`"`, "the lock")
			})

			t.Run("adds a newline before a local file when the rendering lacks one", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+"notes.txt", "second\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "notes.txt", Class: language.Managed, Content: []byte("first")},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, "notes.txt"), "first\nsecond\n", "the extended notes.txt")
			})

			t.Run("appends a local file to an empty rendering", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+"notes.txt", "only\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "notes.txt", Class: language.Managed, Content: []byte{}},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, "notes.txt"), "only\n", "the extended notes.txt")
			})

			tests := []struct {
				name  string
				local string
			}{
				{
					name:  "returns ErrUnmanagedLocal for a local file of a path that is not managed",
					local: localDir + "other.txt",
				},
				{
					name:  "returns ErrUnmanagedLocal for a local file of a seeded file",
					local: localDir + readme,
				},
				{
					name:  "returns ErrUnmanagedLocal for a local directory that is a file",
					local: strings.TrimSuffix(localDir, "/"),
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					root := directory(t)
					put(t, root, tt.local, "local\n")
					_, err := repository(t, root).New(answers(), baseline.Options{})
					assert.ErrorIs(t, err, baseline.ErrUnmanagedLocal, "New")
				})
			}

			t.Run("returns an error for a local directory that does not read", func(t *testing.T) {
				t.Parallel()
				fsys := fault(t, opOpen, localDir+"sub")
				put(t, fsys.Root, localDir+"sub/"+ignore, "local/\n")
				_, err := repository(t, fsys).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
			})

			t.Run("returns an error for a local file that does not read", func(t *testing.T) {
				t.Parallel()
				fsys := fault(t, opOpen, localDir+ignore)
				put(t, fsys.Root, localDir+ignore, "local/\n")
				_, err := repository(t, fsys).New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, errFault, "New")
			})
		})

		t.Run("Add", func(t *testing.T) {
			t.Parallel()

			t.Run("returns an error when the lock does not read before the write of the lock", func(t *testing.T) {
				t.Parallel()
				fsys := fault(t, opRead, lockPath)
				fsys.after = 3
				r := repository(t, fsys)
				_, err := r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				_, err = r.Add([]workspace.Language{beta}, baseline.Options{})
				assert.ErrorIs(t, err, errFault, "Add")
			})
		})

		t.Run("Remove", func(t *testing.T) {
			t.Parallel()

			t.Run("returns an error when a file cannot be removed", func(t *testing.T) {
				t.Parallel()
				r := repository(t, fault(t, opRemove, alphaCfg))
				_, err := r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				_, err = r.Remove([]workspace.Language{alpha})
				assert.ErrorIs(t, err, errFault, "Remove")
			})

			t.Run("returns an error for a recorded path that is a directory", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				assert.NoError(t, root.Remove(alphaCfg), "Remove of the file of alpha")
				assert.NoError(t, root.Mkdir(alphaCfg, 0o755), "Mkdir of the path of the file of alpha")
				_, err := r.Remove([]workspace.Language{alpha})
				assert.HasError(t, err, "Remove")
			})
		})

		t.Run("Sync", func(t *testing.T) {
			t.Parallel()

			t.Run("returns an error for a managed path that is a directory", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
				assert.NoError(t, root.Mkdir(license, 0o755), "Mkdir of LICENSE")
				_, err := r.Sync(nil, baseline.Options{})
				assert.HasError(t, err, "Sync")
				assert.Contains(t, err.Error(), "baseline: read LICENSE", "the error")
			})

			t.Run("returns the error of a write before the conflicts", func(t *testing.T) {
				t.Parallel()
				fsys := fault(t, opWrite, license)
				r := repository(t, fsys)
				fsys.op = ""
				_, err := r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				put(t, fsys.Root, ignore, "# edited by hand\n")
				fsys.op = opWrite
				_, err = r.Sync(func(a *language.Answers) { a.Owner = "Other B.V." }, baseline.Options{})
				assert.ErrorIs(t, err, errFault, "Sync")
			})
		})

		t.Run("Check", func(t *testing.T) {
			t.Parallel()

			t.Run("returns an error for a lock that does not read", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				assert.NoError(t, root.MkdirAll(lockPath, 0o755), "MkdirAll of the lock")
				_, err := repository(t, root).Check()
				assert.HasError(t, err, "Check")
			})

			t.Run("returns an error for a managed path that is a directory", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
				assert.NoError(t, root.Mkdir(license, 0o755), "Mkdir of LICENSE")
				_, err := r.Check()
				assert.HasError(t, err, "Check")
			})

			t.Run("returns an error for a recorded path that is a directory", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
				assert.NoError(t, root.Mkdir(license, 0o755), "Mkdir of LICENSE")
				r, err := baseline.Open(root, catalog(t), version)
				assert.NoError(t, err, "Open without common")
				_, err = r.Check()
				assert.HasError(t, err, "Check")
			})
		})
	})
}

// fault returns a faulty file system of a new temporary directory that fails op on name.
func fault(t *testing.T, op, name string) *faulty {
	t.Helper()
	return &faulty{Root: directory(t), op: op, name: name}
}
