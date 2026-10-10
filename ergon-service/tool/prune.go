// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
)

// moduleTree is the directory of the cache under which the runner installs the Go modules.
const moduleTree = "module"

// Prune removes each file and directory of the cache that is not the install of a tool of sections
// at the version that its options name. sections maps the name of each section to its options.
// Prune returns the removed paths, relative to the cache, in the order of a walk of the cache.
// These are the earlier versions of the tools, the programs of a Go module for another version of
// the go command, the programs of golangci-lint for other module plugins, and the files of an
// install that did not finish. Prune also removes the tools of another repository that shares the
// cache.
//
// The install of a tool is what [Runner.Run] writes for it: the program of a release binary and of
// a Go module, the directory of the program that golangci-lint custom builds with the plugins of
// its section, the root of a crate, the jar of a Maven artifact, and the project of the Composer
// packages of a section. A tool that runs through uv or npx has no install, because uv and npx
// keep their own caches. When go env GOVERSION in the directory Dir does not report a version,
// Prune keeps every Go module of the cache, because their programs depend on the version of the go
// command.
//
// Prune walks and removes through an [os.Root] of the cache, so a symbolic link in the cache does
// not lead a removal outside it. A cache that does not exist has nothing to remove. Prune must not
// run while another process installs a tool into the cache.
//
// It returns an error that wraps [ErrInstall] when a section has no plugins at the key of the tag
// plugins of a tool. It returns the error of a file that it cannot remove, with the paths that it
// removed before the error.
func (r *Runner) Prune(ctx context.Context, sections map[string]language.Options) ([]string, error) {
	cache := filepath.Clean(r.Cache)
	var goVersion []byte
	asked := false
	version := func() []byte {
		if !asked {
			goVersion, _ = r.goVersion(ctx)
			asked = true
		}
		return goVersion
	}
	keep, parents := map[string]bool{}, map[string]bool{}
	for _, section := range slices.Sorted(maps.Keys(sections)) {
		o := sections[section]
		for _, name := range toolNames(o) {
			e, err := lookup(o, section, name)
			if err != nil {
				return nil, err
			}
			for _, p := range r.installed(cache, section, &e, version) {
				// Every path of installed is below the cache.
				rel, _ := filepath.Rel(cache, p)
				install := filepath.ToSlash(rel)
				keep[install] = true
				for dir := path.Dir(install); dir != "." && !parents[dir]; dir = path.Dir(dir) {
					parents[dir] = true
				}
			}
		}
	}
	root, err := os.OpenRoot(cache)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tool: open the cache: %w", err)
	}
	var removed []string
	err = fs.WalkDir(root.FS(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || name == "." || parents[name] {
			return err
		}
		if !keep[name] {
			if err := root.RemoveAll(name); err != nil {
				return fmt.Errorf("tool: remove %s from the cache: %w", name, err)
			}
			removed = append(removed, filepath.FromSlash(name))
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	return removed, errors.Join(err, root.Close())
}

// installed returns the paths below the cache directory root of the install of the tool e of
// section, as [Runner.Prune] states them. It returns none for a tool without an install. version
// returns the version of the go command, or nil when go env GOVERSION does not report a version.
// For a nil version, installed returns root/module for a Go module. It does not return a path for a
// release binary on a platform without an asset or a digest, because [Runner.Run] does not install
// it there. For the release of uv of a section, it returns the program under the name of the tool
// and the program under the name uv, under which [Runner.Run] installs it for a PyPI package.
func (r *Runner) installed(root, section string, e *entry, version func() []byte) []string {
	if rel, ok := reflect.TypeAssert[option.Release](e.value); ok {
		names := []string{e.name}
		if _, uv := reflect.TypeAssert[option.UV](e.value); uv {
			names = append(names, uvName)
		}
		var programs []string
		for _, n := range names {
			if program, _, err := r.releaseProgram(rel, n); err == nil {
				programs = append(programs, program)
			}
		}
		return programs
	}
	switch tool := e.value.Interface().(type) {
	case option.Module:
		goVersion := version()
		if goVersion == nil {
			return []string{filepath.Join(root, moduleTree)}
		}
		name := e.program(goProgram(tool.Package()))
		program := filepath.Join(r.moduleDir(tool, goVersion), r.executable(name))
		if len(e.plugins) == 0 {
			return []string{program}
		}
		dir, _ := customDir(program, tool, e.plugins, name)
		return []string{program, dir}
	case option.Crate:
		return []string{filepath.Join(r.Cache, "crate", tool.Package(), tool.Version(), r.platform())}
	case option.Maven:
		_, jar := r.mavenJar(tool, e.field.Tag.Get(option.ClassifierTag))
		return []string{jar}
	case option.Composer:
		dir, _ := r.composerProject(section, e.tools)
		return []string{dir}
	default:
		return nil
	}
}

// toolNames returns the yaml keys of the exported fields of the group tools of o, in the order of
// the fields, and none for options without the group.
func toolNames(o language.Options) []string {
	v := reflect.ValueOf(o)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	tools, ok := field(v.Elem(), toolsKey)
	if !ok || tools.value.Kind() != reflect.Struct {
		return nil
	}
	var names []string
	for i := range tools.value.NumField() {
		f := tools.value.Type().Field(i)
		if key, _, _ := strings.Cut(f.Tag.Get(yamlTag), ","); key != "" && f.IsExported() {
			names = append(names, key)
		}
	}
	return names
}
