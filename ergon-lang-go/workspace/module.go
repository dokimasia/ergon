// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// The files of the go command.
const (
	// ModFile is the file that declares a module, in the directory of the module.
	ModFile = "go.mod"

	// SumFile is the file of the hashes of the modules that a module requires, beside its go.mod.
	SumFile = "go.sum"

	// WorkFile is the file that lists the modules of a workspace, at the root of the repository.
	WorkFile = "go.work"
)

// ErrModules is the error for a go.work or a go.mod that the go command refuses, and for a directory
// of go.work without a go.mod.
var ErrModules = errors.New("workspace: invalid modules")

// Module is a Go module of a repository.
type Module struct {
	// File is the go.mod of the module, as golang.org/x/mod/modfile parses it.
	File *modfile.File

	// Path is the module path that go.mod declares.
	Path string

	// Dir is the directory of the module, relative to the root of the repository and
	// slash-separated: "." for the module at the root.
	Dir string
}

// Modules returns the modules of the repository at root: the module in each directory that go.work
// uses, in the order of go.work, or the module at root for a repository without go.work. It skips a
// directory of go.work outside root, such as ../other, which belongs to another repository, and
// returns no module for a repository without go.work and without a go.mod at root. It reads the
// files through an [os.Root] of root, so a file that a link outside root replaces fails.
//
// It returns an error that wraps [ErrModules] for a go.work or a go.mod that the go command
// refuses, with the file and the line, and for a directory of go.work without a go.mod. It returns
// the error of opening root, and the error of reading a file, with the file.
func Modules(root string) ([]Module, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	return modules(r)
}

// modules returns the modules of the repository of r, as [Modules] states.
func modules(r *os.Root) ([]Module, error) {
	data, err := r.ReadFile(WorkFile)
	var dirs []string
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if _, statErr := r.Stat(ModFile); errors.Is(statErr, fs.ErrNotExist) {
			return nil, nil
		}
		dirs = []string{"."}
	case err != nil:
		return nil, fmt.Errorf("workspace: read %s: %w", WorkFile, err)
	default:
		if dirs, err = uses(data); err != nil {
			return nil, err
		}
	}
	mods := make([]Module, 0, len(dirs))
	for _, dir := range dirs {
		m, err := read(r, dir)
		if err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// uses returns the directories that the go.work in data uses, relative to the root of the
// repository and slash-separated, in the order of go.work, without a directory outside the root. It
// returns an error that wraps [ErrModules] for a go.work that the go command refuses.
func uses(data []byte) ([]string, error) {
	work, err := modfile.ParseWork(WorkFile, data, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrModules, err)
	}
	dirs := make([]string, 0, len(work.Use))
	for _, use := range work.Use {
		dir := path.Clean(filepath.ToSlash(use.Path))
		if !filepath.IsAbs(use.Path) && dir != ".." && !strings.HasPrefix(dir, "../") {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

// read returns the module in the directory dir of the repository of r. It returns an error that
// wraps [ErrModules] for a go.mod that the go command refuses or that declares no module, and for a
// dir without a go.mod.
func read(r *os.Root, dir string) (Module, error) {
	file := path.Join(dir, ModFile)
	data, err := r.ReadFile(filepath.FromSlash(file))
	if errors.Is(err, fs.ErrNotExist) {
		return Module{}, fmt.Errorf("%w: the directory %s, which go.work uses, has no %s", ErrModules, dir, ModFile)
	}
	if err != nil {
		return Module{}, fmt.Errorf("workspace: read %s: %w", file, err)
	}
	f, err := modfile.Parse(file, data, nil)
	if err != nil {
		return Module{}, fmt.Errorf("%w: %w", ErrModules, err)
	}
	if f.Module == nil {
		return Module{}, fmt.Errorf("%w: %s declares no module", ErrModules, file)
	}
	return Module{File: f, Path: f.Module.Mod.Path, Dir: dir}, nil
}
