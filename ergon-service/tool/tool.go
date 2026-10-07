// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
)

// The errors of [Runner.Run].
var (
	// ErrUnknown is the error for a tool that the section does not name. The command exits 2 for it.
	ErrUnknown = errors.New("tool: the section names no such tool")

	// ErrInstall is the error for a tool that does not install: a platform without an asset or a
	// digest, a download that fails or that differs from its digest, an archive without the
	// program, a toolchain that is missing or that fails, and a section without the uv of its PyPI
	// packages.
	ErrInstall = errors.New("tool: the tool does not install")
)

// The keys of the options that the runner reads: the group of the tools of a section, and the
// key of a field in its yaml tag.
const (
	toolsKey = "tools"
	yamlTag  = "yaml"
)

// windows is the system whose programs end in .exe.
const windows = "windows"

// major matches the last element of the path of a Go module of a major version, such as v2, which
// go install does not name the program after.
var major = regexp.MustCompile(`^v[0-9]+$`)

// Runner installs the tools of the sections of .ergon.yaml into its cache, and runs them. Each
// field is required, and the zero value runs nothing.
//
// # Concurrency
//
// A Runner is safe for concurrent use. Two processes that install one tool at a time each write
// the program to a temporary file of the cache and rename it, so a run reads a whole program.
type Runner struct {
	// Client downloads the release binaries and the jars of Maven.
	Client *http.Client

	// Stdin is the standard input of a tool.
	Stdin io.Reader

	// Stdout is the standard output of a tool and of a toolchain that installs one.
	Stdout io.Writer

	// Stderr is the standard error of a tool and of a toolchain that installs one.
	Stderr io.Writer

	// Cache is the directory of the installed tools, such as ergon/tools in the cache directory of
	// the user.
	Cache string

	// Dir is the working directory of a tool: the working directory of the command that runs it,
	// such as a module of Go in a repository of several modules.
	Dir string

	// Platform is the platform that the runner installs for, such as linux/amd64.
	Platform option.Platform

	// Env is the environment of a tool and of a toolchain, such as the environment of the process.
	Env []string
}

// entry is a tool of the struct of the tools of a section.
type entry struct {
	// value is the value of the tool.
	value reflect.Value

	// tools is the struct of the tools of the section, with the uv of its PyPI packages and its
	// Composer packages.
	tools reflect.Value

	// name is the name of the tool, its key in the section.
	name string

	// field is the field of the tool, with its tags.
	field reflect.StructField
}

// Run installs the tool name of the options o of the section section, unless the cache has it, and
// runs it with args in the directory Dir. It returns the exit status of the tool.
//
// The type of the tool states how it installs and runs:
//
//   - a [option.Release] downloads the asset of the platform, which Run checks against the digest
//     of the pin before it unpacks the program
//   - a [option.Module] installs with go install
//   - a [option.PyPI] runs through the [option.UV] of the section, with uv tool run, or with uv run
//     in the environment of the project for a tool tagged run:"project"
//   - an [option.NPM] runs with npx
//   - an [option.Crate] installs with cargo install --locked
//   - a [option.Maven] downloads its jar, which Run checks against the .sha256 file beside it, and
//     runs with java -jar
//   - an [option.Composer] installs with every Composer package of the section into one project,
//     and runs with php
//
// It returns an error that wraps [ErrUnknown] for options without the tool name, which lists the
// tools of the section, [ErrInstall] for a tool that does not install, and the error of a tool that
// does not start. A tool that runs and fails returns its exit status and no error.
func (r *Runner) Run(ctx context.Context, section string, o language.Options, name string, args []string) (
	int, error,
) {
	e, err := lookup(o, section, name)
	if err != nil {
		return 0, err
	}
	program, prefix, err := r.install(ctx, section, &e)
	if err != nil {
		return 0, err
	}
	cmd := exec.CommandContext(ctx, program, slices.Concat(prefix, args)...)
	cmd.Dir, cmd.Env = r.Dir, r.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r.Stdin, r.Stdout, r.Stderr
	err = cmd.Run()
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return exit.ExitCode(), nil
	}
	if err != nil {
		return 0, fmt.Errorf("tool: run %s.%s: %w", section, name, err)
	}
	return 0, nil
}

// install installs the tool e of section, unless the cache has it, and returns the program that
// runs it and the arguments before the arguments of the tool. It returns an error that wraps
// [ErrInstall] for a tool that does not install, and for a field of a type that is no kind of tool.
func (r *Runner) install(ctx context.Context, section string, e *entry) (string, []string, error) {
	if release, ok := reflect.TypeAssert[option.Release](e.value); ok {
		program, err := r.release(ctx, release, e.name)
		return program, nil, err
	}
	switch tool := e.value.Interface().(type) {
	case option.Module:
		program, err := r.module(ctx, tool, e.program(goProgram(tool.Package())))
		return program, nil, err
	case option.PyPI:
		return r.pypi(ctx, e, tool)
	case option.NPM:
		program, args := npm(tool, e.program(path.Base(tool.Package())))
		return program, args, nil
	case option.Crate:
		program, err := r.crate(ctx, tool, e.program(tool.Package()))
		return program, nil, err
	case option.Maven:
		return r.maven(ctx, tool, e.field.Tag.Get(option.ClassifierTag))
	case option.Composer:
		return r.composer(ctx, section, e, e.program(path.Base(tool.Package())))
	default:
		return "", nil, fmt.Errorf("%w: %s.%s is a %s, which is no kind of tool", ErrInstall, section, e.name,
			e.value.Type())
	}
}

// program returns the program of the tool e: the program of its tag program, and fallback for a
// tool without the tag.
func (e *entry) program(fallback string) string {
	if program := e.field.Tag.Get(option.ProgramTag); program != "" {
		return program
	}
	return fallback
}

// lookup returns the tool name of the struct of the tools of o, the options of section: the field
// of the group tools whose yaml key is name. It returns an error that wraps [ErrUnknown] for options
// without the group, and for a group without the key, which lists the tools of the group as
// <section>.<tool>.
func lookup(o language.Options, section, name string) (entry, error) {
	v := reflect.ValueOf(o)
	var tools entry
	ok := v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Struct
	if ok {
		tools, ok = field(v.Elem(), toolsKey)
	}
	if !ok || tools.value.Kind() != reflect.Struct {
		return entry{}, fmt.Errorf("%w: %s.%s, because the section %s has no tools", ErrUnknown, section, name,
			section)
	}
	tool, ok := field(tools.value, name)
	if !ok {
		var names []string
		for i := range tools.value.NumField() {
			f := tools.value.Type().Field(i)
			if key, _, _ := strings.Cut(f.Tag.Get(yamlTag), ","); key != "" && f.IsExported() {
				names = append(names, section+"."+key)
			}
		}
		return entry{}, fmt.Errorf("%w: %s.%s, which is none of %s", ErrUnknown, section, name,
			strings.Join(names, ", "))
	}
	return entry{value: tool.value, tools: tools.value, name: name, field: tool.field}, nil
}

// field returns the field of the struct s whose yaml key is key, and reports whether s has one.
func field(s reflect.Value, key string) (entry, bool) {
	for i := range s.NumField() {
		f := s.Type().Field(i)
		if name, _, _ := strings.Cut(f.Tag.Get(yamlTag), ","); name == key && f.IsExported() {
			return entry{value: s.Field(i), field: f}, true
		}
	}
	return entry{}, false
}

// goProgram returns the name of the program that go install writes for the package pkg: its last
// element, or the element before it for a path that ends in a major version, such as v2.
func goProgram(pkg string) string {
	base := path.Base(pkg)
	if major.MatchString(base) {
		return path.Base(path.Dir(pkg))
	}
	return base
}
