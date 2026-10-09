// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/option"
)

// uvName is the name of the release binary of uv, which runs the PyPI packages of a section.
const uvName = "uv"

// project is the composer.json of the project of the Composer packages of a section, before
// composer require adds the packages: a project that allows the plugins of its packages.
const project = `{"config": {"allow-plugins": true}}` + "\n"

// filePerm is the mode of a file of the cache that is no program.
const filePerm fs.FileMode = 0o644

// module installs the Go module m with go install into the cache, unless the cache has its program
// name, and returns the program. The cache keeps a program for each version of the go command in
// the directory Dir, as go env GOVERSION reports it, because a program that reads the packages of
// Go can refuse a go command of another version, as dokimi-mutate-go refuses the export data of
// one. go install checks the module against the checksum database. It returns an error that wraps
// [ErrInstall] for a go command that does not report its version.
func (r *Runner) module(ctx context.Context, m option.Module, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOVERSION")
	cmd.Dir, cmd.Env, cmd.Stderr = r.Dir, r.Env, r.Stderr
	goVersion, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: go env GOVERSION: %w", ErrInstall, err)
	}
	sum := sha256.Sum256(bytes.TrimSpace(goVersion))
	dir := filepath.Join(r.Cache, "module", filepath.FromSlash(m.Package()), m.Version(),
		hex.EncodeToString(sum[:8]), r.platform())
	program := filepath.Join(dir, r.executable(name))
	if exists(program) {
		return program, nil
	}
	return program, r.toolchain(ctx, []string{"GOBIN=" + dir}, "go", "install", string(m))
}

// crate installs the crate c with cargo install --locked into the cache, unless the cache has its
// program name, and returns the program.
func (r *Runner) crate(ctx context.Context, c option.Crate, name string) (string, error) {
	root := filepath.Join(r.Cache, "crate", c.Package(), c.Version(), r.platform())
	program := filepath.Join(root, "bin", r.executable(name))
	if exists(program) {
		return program, nil
	}
	return program, r.toolchain(ctx, nil, "cargo", "install", "--locked", "--root", root, string(c))
}

// pypi installs the uv of the section of e, and returns uv and the arguments that run the PyPI
// package p: uv tool run in an environment of its own, or uv run with the package in the
// environment of the project for a tool tagged run:"project". It returns an error that wraps
// [ErrInstall] for a section without a uv, and the error of the installation of uv.
func (r *Runner) pypi(ctx context.Context, e *entry, p option.PyPI) (string, []string, error) {
	var uv option.UV
	found := false
	for i := range e.tools.NumField() {
		if f := e.tools.Type().Field(i); f.IsExported() && f.Type == reflect.TypeFor[option.UV]() {
			uv, found = reflect.TypeAssert[option.UV](e.tools.Field(i))
		}
	}
	if !found {
		return "", nil, fmt.Errorf("%w: %s, whose section has no uv", ErrInstall, e.name)
	}
	program, err := r.release(ctx, uv, uvName)
	if err != nil {
		return "", nil, err
	}
	spec := p.Package() + "==" + p.Version()
	name := e.program(p.Package())
	if e.field.Tag.Get(option.RunTag) == option.Project {
		return program, []string{"run", "--with", spec, "--", name}, nil
	}
	return program, []string{"tool", "run", "--from", spec, name}, nil
}

// npm returns npx and the arguments that run the program name of the npm package n, which npx
// installs into its cache.
func npm(n option.NPM, name string) (string, []string) {
	return "npx", []string{"--yes", "--package=" + string(n), "--", name}
}

// composer installs every Composer package of the section of e together into one project of the
// cache, with composer require, unless the cache has the program name, and returns php and the
// arguments that run the program. The project is named after the section and the digest of its
// packages, so a section with other packages or versions installs a new project. The project
// allows the plugins of its packages, which the section pins, so a plugin such as the extension
// installer of PHPStan loads the extensions of the project. It returns an error that wraps
// [ErrInstall] for a project that does not create, and the error of composer.
func (r *Runner) composer(ctx context.Context, section string, e *entry, name string) (string, []string, error) {
	var packages []string
	for i := range e.tools.NumField() {
		if !e.tools.Type().Field(i).IsExported() {
			continue
		}
		if c, ok := reflect.TypeAssert[option.Composer](e.tools.Field(i)); ok {
			packages = append(packages, c.Package()+":"+c.Version())
		}
	}
	slices.Sort(packages)
	sum := sha256.Sum256([]byte(strings.Join(packages, "\n")))
	dir := filepath.Join(r.Cache, "composer", section, hex.EncodeToString(sum[:8]))
	program := filepath.Join(dir, "vendor", "bin", name)
	if exists(program) {
		return "php", []string{program}, nil
	}
	err := os.MkdirAll(dir, dirPerm)
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, "composer.json"), []byte(project), filePerm)
	}
	if err != nil {
		return "", nil, fmt.Errorf("%w: create the project of composer %s: %w", ErrInstall, dir, err)
	}
	args := slices.Concat([]string{"require", "--no-interaction", "--no-progress", "--working-dir=" + dir}, packages)
	if err := r.toolchain(ctx, nil, "composer", args...); err != nil {
		return "", nil, err
	}
	return "php", []string{program}, nil
}

// toolchain runs the program name of a toolchain with args and the environment of the runner and
// env, and writes its output to the standard error of the runner, so the output of the tool stays
// apart. It returns an error that wraps [ErrInstall] for a program that does not start or that
// fails.
func (r *Runner) toolchain(ctx context.Context, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = r.Dir, slices.Concat(r.Env, env)
	cmd.Stdout, cmd.Stderr = r.Stderr, r.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s %s: %w", ErrInstall, name, strings.Join(args, " "), err)
	}
	return nil
}

// platform returns the platform of the runner as the name of a directory, such as linux-amd64.
func (r *Runner) platform() string {
	return strings.ReplaceAll(string(r.Platform), "/", "-")
}

// executable returns the name of the program name on the platform of the runner: name with .exe on
// Windows, and name on any other system.
func (r *Runner) executable(name string) string {
	if r.Platform.OS() == windows {
		return name + ".exe"
	}
	return name
}

// exists reports whether name exists.
func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
