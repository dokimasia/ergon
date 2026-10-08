// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// ErrSource is the error of [Find] for a source tag outside the forms chocolatey:<package> and
// github:<owner>/<name>.
var ErrSource = errors.New("pin: invalid source tag")

// The registries of a source tag.
const (
	chocolateySource = "chocolatey"
	githubSource     = "github"
)

// Kind is the registry that publishes the releases of a pin. The zero value is not a valid kind.
type Kind uint8

// The kinds of pin.
const (
	// KindModule is a Go module of the module proxy.
	KindModule Kind = 1

	// KindPyPI is a package of PyPI.
	KindPyPI Kind = 2

	// KindNPM is a package of the npm registry.
	KindNPM Kind = 3

	// KindCrate is a crate of crates.io.
	KindCrate Kind = 4

	// KindMaven is an artifact of Maven Central.
	KindMaven Kind = 5

	// KindComposer is a package of Packagist.
	KindComposer Kind = 6

	// KindBinary is a release binary of the GitHub releases of its repository.
	KindBinary Kind = 7

	// KindAction is an action of the GitHub releases of its repository.
	KindAction Kind = 8

	// KindChocolatey is a package of the Chocolatey community repository.
	KindChocolatey Kind = 9

	// KindGitHub is a version of the GitHub releases of a repository, such as the release of the
	// hooks of pre-commit.
	KindGitHub Kind = 10
)

// Valid reports whether k is one of the kinds.
func (k Kind) Valid() bool {
	return k >= KindModule && k <= KindGitHub
}

// Pin is a field of the options of a producer whose value refers to a release of a registry.
type Pin struct {
	// Value is the value of the field, such as an option.Module or a workflow.Action.
	Value any

	// Key is the key of the field in .ergon.yaml, such as go.tools.golangci-lint.
	Key string

	// Name is the released project in its registry: a package path of Go, a package, a crate,
	// group:artifact, vendor/package, a repository as owner/name, or a package of Chocolatey.
	Name string

	// Version is the version of the pin, such as v2.14.0 or 0.12.0.
	Version string

	// Field are the names of the Go fields from the options struct to the pin, such as Tools and
	// GolangCILint. An embedded struct has the name of its type.
	Field []string

	// Kind is the registry of the pin.
	Kind Kind
}

// Find returns the pins of o, the options of the section section, in the order in which
// [options.Fields] returns their keys. A key is a pin by the type of its field:
//
//   - a release binary, a type that implements [option.Release], of the releases of its repository
//   - an [option.Module], [option.PyPI], [option.NPM], [option.Crate], [option.Maven] or
//     [option.Composer] of its registry
//   - a [workflow.Action], of the releases of the repository of its Uses
//   - an [option.Version] with an [option.SourceTag], of the registry that the tag names
//
// The key of a pin is section, a dot and its key in the section. Find returns an error that wraps
// [options.ErrDefect] for options that [options.Fields] refuses, and [ErrSource] for a source tag
// that is not chocolatey:<package> or github:<owner>/<name>.
func Find(section string, o language.Options) ([]Pin, error) {
	fs, err := options.Fields(o)
	if err != nil {
		return nil, fmt.Errorf("pin: find the pins of %s: %w", section, err)
	}
	s := reflect.ValueOf(o).Elem()
	var pins []Pin
	for _, f := range fs {
		p, ok, err := classify(s.FieldByIndex(f.Index), s.Type().FieldByIndex(f.Index).Tag, section+"."+f.Key)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		p.Field = make([]string, len(f.Index))
		for i := range f.Index {
			p.Field[i] = s.Type().FieldByIndex(f.Index[:i+1]).Name
		}
		pins = append(pins, p)
	}
	return pins, nil
}

// classify returns the pin of v, the value of a field whose struct tag is tag and whose key is key,
// and reports whether v is a pin, as [Find] states. It returns an error that wraps [ErrSource] for a
// source tag that is not chocolatey:<package> or github:<owner>/<name>.
func classify(v reflect.Value, tag reflect.StructTag, key string) (Pin, bool, error) {
	p := Pin{Value: v.Interface(), Key: key}
	if release, ok := reflect.TypeAssert[option.Release](v); ok {
		p.Kind, p.Name, p.Version = KindBinary, release.Repository(), release.Pin().Version
		return p, true, nil
	}
	switch value := p.Value.(type) {
	case option.Module:
		p.Kind, p.Name, p.Version = KindModule, value.Package(), value.Version()
	case option.PyPI:
		p.Kind, p.Name, p.Version = KindPyPI, value.Package(), value.Version()
	case option.NPM:
		p.Kind, p.Name, p.Version = KindNPM, value.Package(), value.Version()
	case option.Crate:
		p.Kind, p.Name, p.Version = KindCrate, value.Package(), value.Version()
	case option.Maven:
		p.Kind, p.Name, p.Version = KindMaven, value.Package(), value.Version()
	case option.Composer:
		p.Kind, p.Name, p.Version = KindComposer, value.Package(), value.Version()
	case workflow.Action:
		owner, rest, _ := strings.Cut(value.Uses, "/")
		name, _, _ := strings.Cut(rest, "/")
		p.Kind, p.Name, p.Version = KindAction, owner+"/"+name, value.Release
	case option.Version:
		source, ok := tag.Lookup(option.SourceTag)
		if !ok {
			return Pin{}, false, nil
		}
		registry, name, _ := strings.Cut(source, ":")
		switch {
		case registry == chocolateySource && name != "":
			p.Kind = KindChocolatey
		case registry == githubSource && language.Repository(name).Valid():
			p.Kind = KindGitHub
		default:
			return Pin{}, false, fmt.Errorf("%w: %s, whose source %q is not chocolatey:<package> or "+
				"github:<owner>/<name>", ErrSource, key, source)
		}
		p.Name, p.Version = name, string(value)
	default:
		return Pin{}, false, nil
	}
	return p, true, nil
}
