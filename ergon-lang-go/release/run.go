// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	goworkspace "go.dokimi.dev/ergon/lang/go/workspace"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"
)

// The flags of the go command against the proxy of a run. go mod tidy may write go.mod and go.sum,
// while a build of [Packer.Pack] takes them as they are. Both leave the module cache writable, so
// the temporary directory of the run can be removed.
const (
	tidyFlags  = "-mod=mod -modcacherw"
	buildFlags = "-modcacherw"
)

// modcacheDir is the directory of the module cache of a run in its temporary directory.
const modcacheDir = "modcache"

// run is one call of [Versioner.Apply], [Locker.Stale], [Locker.Lock] or [Packer.Pack]: the modules
// of the repository, the versions that it releases, the files that it changes, and the proxy that
// serves the released modules to the go command.
type run struct {
	// snapshot returns the tree of the working tree of a directory as git would commit it.
	snapshot func(ctx context.Context, dir string) (string, error)

	// modules maps the path of each module of the repository to the module, with its go.mod as
	// Apply rewrote it.
	modules map[string]*goworkspace.Module

	// released maps the path of each module that the call releases to its version, with a v before
	// it: the new version of an edit of Apply, and the version of a package of Stale and Lock.
	released map[string]string

	// served reports the paths of the released modules that the proxy serves.
	served map[string]bool

	// proxy serves the released modules.
	proxy *proxy

	// work is the go.work of the repository, or nil for a repository without one.
	work *modfile.WorkFile

	// root is the root of the repository.
	root string

	// tree is the snapshot of the working tree since the last go mod tidy, or empty when the next
	// zip needs a new one.
	tree string

	// env are the variables of the environment of go mod tidy, after the variables of the process.
	env []string

	// files are the files that the call changes.
	files changes
}

// newRun returns a run in the repository at root that takes its snapshots with snapshot and
// releases no module yet. It returns the error of reading and of parsing go.work, and the error of
// [goworkspace.Modules].
func newRun(snapshot func(ctx context.Context, dir string) (string, error), root string) (*run, error) {
	work, err := readWork(root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	mods, err := goworkspace.Modules(root)
	if err != nil {
		return nil, err
	}
	r := &run{
		snapshot: snapshot,
		modules:  make(map[string]*goworkspace.Module, len(mods)),
		released: map[string]string{},
		served:   map[string]bool{},
		work:     work,
		root:     root,
		files:    changes{before: map[string][]byte{}, root: root},
	}
	for i := range mods {
		r.modules[mods[i].Path] = &mods[i]
	}
	return r, nil
}

// checkModule returns an error for a name that is the path of no module of the repository.
func (r *run) checkModule(name string) error {
	if r.modules[name] == nil {
		return fmt.Errorf("release: the module %s, which the repository does not have", name)
	}
	return nil
}

// release writes the requirements of edits into the go.mod of their modules and the replaces of
// go.work, and runs go mod tidy in each module that requires a rewritten module without a directory
// replace, as [run.tidyAll] runs it. It returns the error of a write and of tidyAll.
func (r *run) release(ctx context.Context, edits []language.Edit) error {
	var pending []*goworkspace.Module
	rewritten := false
	for k := range edits {
		e := &edits[k]
		if len(e.Requirements) == 0 {
			continue
		}
		m := r.modules[e.Package.Name]
		tidy := false
		for _, d := range e.Requirements {
			// AddRequire returns no error.
			_ = m.File.AddRequire(d.Name, d.Req)
			tidy = tidy || !replaced(m.File, d.Name)
		}
		m.File.Cleanup()
		if err := r.files.write(path.Join(m.Dir, goworkspace.ModFile), modfile.Format(m.File.Syntax)); err != nil {
			return err
		}
		rewritten = true
		if tidy {
			pending = append(pending, m)
		}
	}
	if rewritten {
		if err := r.pin(); err != nil {
			return err
		}
	}
	return r.tidyAll(ctx, pending)
}

// tidyAll runs go mod tidy in each module of pending, a module after every released module of
// pending that it requires, against a module proxy in a temporary directory, which it removes
// afterwards. It does nothing for no pending module. It returns an error that wraps [ErrCycle] for
// modules of pending that require each other, and the error of creating the proxy, of go env and of
// [run.tidy].
func (r *run) tidyAll(ctx context.Context, pending []*goworkspace.Module) error {
	if len(pending) == 0 {
		return nil
	}
	tmp, err := r.open()
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if r.env, err = r.environment(ctx, filepath.Join(tmp, modcacheDir), tidyFlags); err != nil {
		return err
	}
	for len(pending) > 0 {
		at := slices.IndexFunc(pending, func(m *goworkspace.Module) bool { return r.ready(m, pending) })
		if at < 0 {
			paths := make([]string, len(pending))
			for k, m := range pending {
				paths[k] = m.Path
			}
			return fmt.Errorf("%w without a directory replace: %s", ErrCycle, strings.Join(paths, ", "))
		}
		if err := r.tidy(ctx, pending[at]); err != nil {
			return err
		}
		pending = slices.Delete(pending, at, at+1)
	}
	return nil
}

// open creates a temporary directory with the module proxy of r, and returns the directory, which
// the caller removes. It returns the error of creating the directory.
func (r *run) open() (string, error) {
	tmp, err := os.MkdirTemp("", "ergon-release-")
	if err != nil {
		return "", fmt.Errorf("release: create the module proxy: %w", err)
	}
	r.proxy = &proxy{versions: map[string][]string{}, dir: filepath.Join(tmp, "proxy")}
	return tmp, nil
}

// ready reports whether m can be tidied: neither m nor a module of pending other than m is a
// released module that m requires, directly or through other released modules.
func (r *run) ready(m *goworkspace.Module, pending []*goworkspace.Module) bool {
	need := r.closure(m)
	return !slices.ContainsFunc(pending, func(p *goworkspace.Module) bool {
		return slices.Contains(need, p.Path)
	})
}

// closure returns the paths of the released modules that m requires at their new versions without
// a directory replace of its go.mod, directly or through the go.mod of other released modules, in
// the order of a breadth-first search. A released module ignores the replace directives of its
// go.mod, as the go command applies them in the main module alone.
func (r *run) closure(m *goworkspace.Module) []string {
	var need []string
	queue := []*modfile.File{m.File}
	for len(queue) > 0 {
		f := queue[0]
		queue = queue[1:]
		for _, req := range f.Require {
			s := r.modules[req.Mod.Path]
			released := s != nil && r.released[s.Path] == req.Mod.Version
			if !released || slices.Contains(need, s.Path) || (f == m.File && replaced(f, s.Path)) {
				continue
			}
			need = append(need, s.Path)
			queue = append(queue, s.File)
		}
	}
	return need
}

// tidy serves the released modules that m requires from a snapshot of the working tree, and runs
// go mod tidy in m. It returns the error of git, of the proxy and of the go command, with the
// output of the go command.
func (r *run) tidy(ctx context.Context, m *goworkspace.Module) error {
	for _, p := range r.closure(m) {
		if err := r.serve(ctx, r.modules[p]); err != nil {
			return err
		}
	}
	for _, name := range []string{goworkspace.ModFile, goworkspace.SumFile} {
		if err := r.files.touch(path.Join(m.Dir, name)); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = filepath.Join(r.root, filepath.FromSlash(m.Dir))
	cmd.Env = append(os.Environ(), r.env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("release: go mod tidy in %s: %w\n%s", m.Dir, err, out)
	}
	r.tree = ""
	return nil
}

// serve writes the zip of the released module s at its version into the proxy, from a snapshot of
// the working tree, once per run. It returns the error of [run.zip] and of the proxy.
func (r *run) serve(ctx context.Context, s *goworkspace.Module) error {
	if r.served[s.Path] {
		return nil
	}
	m, data, err := r.zip(ctx, s)
	if err != nil {
		return err
	}
	if err := r.proxy.add(m, data, modOf(m, data)); err != nil {
		return err
	}
	r.served[s.Path] = true
	return nil
}

// zip returns the released module s at its version, and its zip as golang.org/x/mod/zip writes it
// from a snapshot of the working tree. It takes the snapshot when r has none since its last go mod
// tidy. It returns the error of the snapshot and of the zip.
func (r *run) zip(ctx context.Context, s *goworkspace.Module) (module.Version, []byte, error) {
	m := module.Version{Path: s.Path, Version: r.released[s.Path]}
	if r.tree == "" {
		tree, err := r.snapshot(ctx, r.root)
		if err != nil {
			return m, nil, err
		}
		r.tree = tree
	}
	subdir := ""
	if s.Dir != "." {
		subdir = s.Dir
	}
	var data bytes.Buffer
	if err := modzip.CreateFromVCS(&data, m, r.root, r.tree, subdir); err != nil {
		return m, nil, fmt.Errorf("release: write the zip of %s: %w", m, err)
	}
	return m, data.Bytes(), nil
}

// environment returns the variables of the environment of the go command against the proxy of r,
// with the module cache in modcache and GOFLAGS set to flags, as [Versioner.Apply] states them. It
// returns the error of go env.
func (r *run) environment(ctx context.Context, modcache, flags string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOPROXY", "GONOSUMDB", "GOPRIVATE", "GOMODCACHE")
	cmd.Dir = r.root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("release: go env: %w", err)
	}
	values := append(strings.Split(string(out), "\n"), "", "", "", "")
	goproxy, nosumdb, private, cache := values[0], values[1], values[2], values[3]
	proxies := []string{r.proxy.url()}
	if cache != "" {
		proxies = append(proxies, (&proxy{dir: filepath.Join(cache, "cache", "download")}).url())
	}
	patterns := []string{cmp.Or(nosumdb, private)}
	for p := range r.modules {
		patterns = append(patterns, p)
	}
	slices.Sort(patterns)
	patterns = slices.DeleteFunc(patterns, func(p string) bool { return p == "" })
	return []string{
		"GOWORK=off",
		"GOFLAGS=" + flags,
		"GOMODCACHE=" + modcache,
		"GOPROXY=" + strings.Join(append(proxies, goproxy), ","),
		"GONOSUMDB=" + strings.Join(patterns, ","),
	}, nil
}

// pin writes go.work with a replace of each version of a module of the repository that a go.mod of
// the repository requires, by the directory of the module, and without a replace of another version
// of such a module. The go command reads the go.mod of every version of a module of the workspace
// that another module requires, so the replaces let the workspace build before the tag of a version
// exists. It writes nothing for a repository without go.work, and returns the error of the write.
func (r *run) pin() error {
	if r.work == nil {
		return nil
	}
	required := map[module.Version]string{}
	for _, m := range r.modules {
		for _, req := range m.File.Require {
			s := r.modules[req.Mod.Path]
			if s == nil || s == m {
				continue
			}
			required[req.Mod] = "./" + s.Dir
			if s.Dir == "." {
				required[req.Mod] = "./"
			}
		}
	}
	// AddReplace puts a new version after a replace of another version of its module, so the new
	// replaces go in before the old ones go out.
	versions := slices.SortedFunc(maps.Keys(required), func(a, b module.Version) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.Version, b.Version))
	})
	for _, v := range versions {
		// AddReplace returns no error.
		_ = r.work.AddReplace(v.Path, v.Version, required[v], "")
	}
	for _, rep := range r.work.Replace {
		if r.modules[rep.Old.Path] != nil && rep.Old.Version != "" && required[rep.Old] == "" {
			// DropReplace returns no error.
			_ = r.work.DropReplace(rep.Old.Path, rep.Old.Version)
		}
	}
	r.work.Cleanup()
	return r.files.write(goworkspace.WorkFile, modfile.Format(r.work.Syntax))
}

// readWork returns the go.work of the repository at root. It returns the error of reading go.work,
// which wraps [fs.ErrNotExist] for a repository without one, and the error of a go.work that the go
// command refuses.
func readWork(root string) (*modfile.WorkFile, error) {
	data, err := os.ReadFile(filepath.Join(root, goworkspace.WorkFile))
	if err != nil {
		return nil, fmt.Errorf("release: read %s: %w", goworkspace.WorkFile, err)
	}
	work, err := modfile.ParseWork(goworkspace.WorkFile, data, nil)
	if err != nil {
		return nil, fmt.Errorf("release: %w", err)
	}
	return work, nil
}

// replaced reports whether f replaces the module path with a directory, which the go command reads
// from disk in the main module.
func replaced(f *modfile.File, path string) bool {
	return slices.ContainsFunc(f.Replace, func(rep *modfile.Replace) bool {
		return rep.Old.Path == path && rep.New.Version == ""
	})
}

// modOf returns the go.mod in data, the zip of the module version m, or the go.mod that the go
// command synthesizes for a module whose zip has none.
func modOf(m module.Version, data []byte) []byte {
	if r, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		if mod, err := fs.ReadFile(r, m.Path+"@"+m.Version+"/"+goworkspace.ModFile); err == nil {
			return mod
		}
	}
	return []byte("module " + modfile.AutoQuote(m.Path) + "\n")
}
