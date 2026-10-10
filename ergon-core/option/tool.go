// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// The rules of the packages of the kinds of tool, and of the versions of Go modules. The
// characters of each rule are plain text in every file that ergon init renders, and in an argument
// of a command. The versions of the other kinds follow the rule of a [Version].
var (
	// modulePath matches the path of a Go module: an element of a domain, then elements of a path.
	modulePath = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+(/[A-Za-z0-9._~-]+)+$`)

	// moduleVersion matches a version of a Go module, which starts with v.
	moduleVersion = regexp.MustCompile(`^v[0-9][A-Za-z0-9.+-]*$`)

	// pypiName matches the name of a PyPI package.
	pypiName = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

	// npmName matches the name of an npm package, with its scope or without one.
	npmName = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._~-]*/)?[a-z0-9][a-z0-9._~-]*$`)

	// crateName matches the name of a crate.
	crateName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

	// mavenName matches the coordinates of a Maven artifact without its version, as
	// group:artifact.
	mavenName = regexp.MustCompile(`^[A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+$`)

	// composerName matches the name of a Composer package, as vendor/package.
	composerName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*/[a-z0-9][a-z0-9_.-]*$`)

	// linterName matches the name of a linter of golangci-lint: a lowercase letter, then lowercase
	// letters, digits, hyphens and underscores.
	linterName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// Module is a Go module and its version, as <module>@<version>, such as
// golang.org/x/vuln/cmd/govulncheck@v1.8.0. ergon tool run installs it with go install, which checks
// it against the checksum database, in the gate of Go alone.
type Module string

// Package returns the path of the module of m, the text before its last @.
func (m Module) Package() string {
	pkg, _ := split(string(m))
	return pkg
}

// Version returns the version of m, the text after its last @.
func (m Module) Version() string {
	_, v := split(string(m))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for an m that is not <module>@<version>: a
// path whose first element is not a domain, or a version that does not start with v and a digit.
func (m Module) Validate() error {
	return validate("<module>@<version>", string(m), modulePath, moduleVersion)
}

// Plugins are the module plugins of golangci-lint, by the name of the linter that each plugin
// registers, such as assertlint: go.dokimi.dev/assert/lint/golangci@v0.1.0. A plugin is the Go
// package that registers the linter, at a version, as <package>@<version>. ergon tool run builds
// the plugins into golangci-lint with golangci-lint custom, whose go get checks each plugin against
// the checksum database.
type Plugins map[string]Module

// Validate returns an error that wraps [ErrInvalid] for a name that is not a lowercase letter
// followed by lowercase letters, digits, hyphens and underscores, and for a plugin that is not
// <module>@<version>, as [Module.Validate] states. golangci-lint refuses a linter whose name has an
// uppercase letter. Validate reads the plugins in the order of their names, so it returns the same
// error for the same plugins.
func (p Plugins) Validate() error {
	for _, name := range slices.Sorted(maps.Keys(p)) {
		if !linterName.MatchString(name) {
			return fmt.Errorf("%w: the linter %q, which is not a lowercase letter followed by lowercase letters, "+
				"digits, hyphens and underscores", ErrInvalid, name)
		}
		if err := p[name].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Linters is a group of options whose module plugins of golangci-lint come from more than one
// option, such as the analyzers of a section beside the plugins that a repository adds. The tag
// plugins of a tool may name such a group in place of an option of the type [Plugins].
type Linters interface {
	// Linters returns the module plugins of the group, by the name of the linter that each
	// registers.
	Linters() Plugins
}

// PyPI is a PyPI package and its version, as <package>@<version>, such as ruff@0.16.10. ergon tool
// run runs it through the release binary of uv of its section, which checks it against the digest
// that PyPI publishes.
type PyPI string

// Package returns the name of the package of p, the text before its last @.
func (p PyPI) Package() string {
	pkg, _ := split(string(p))
	return pkg
}

// Version returns the version of p, the text after its last @.
func (p PyPI) Version() string {
	_, v := split(string(p))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for a p that is not <package>@<version> of
// PyPI.
func (p PyPI) Validate() error {
	return validate("<package>@<version>", string(p), pypiName, version)
}

// NPM is an npm package and its version, as <package>@<version>, such as @biomejs/biome@2.5.15.
// The package of a scope keeps its own @. ergon tool run runs it with npx, in the gates of
// JavaScript and TypeScript alone.
type NPM string

// Package returns the name of the package of n, the text before its last @.
func (n NPM) Package() string {
	pkg, _ := split(string(n))
	return pkg
}

// Version returns the version of n, the text after its last @.
func (n NPM) Version() string {
	_, v := split(string(n))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for an n that is not <package>@<version> of
// npm, with a scope or without one.
func (n NPM) Validate() error {
	return validate("<package>@<version>", string(n), npmName, version)
}

// Crate is a crate and its version, as <crate>@<version>, such as cargo-audit@0.22.2. ergon tool
// run installs it with cargo install --locked, in the gate of Rust alone.
type Crate string

// Package returns the name of the crate of c, the text before its last @.
func (c Crate) Package() string {
	pkg, _ := split(string(c))
	return pkg
}

// Version returns the version of c, the text after its last @.
func (c Crate) Version() string {
	_, v := split(string(c))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for a c that is not <crate>@<version>.
func (c Crate) Validate() error {
	return validate("<crate>@<version>", string(c), crateName, version)
}

// Maven is a Maven artifact and its version, as <group>:<artifact>@<version>, such as
// com.pinterest.ktlint:ktlint-cli@1.8.0. ergon tool run downloads its jar from Maven Central,
// checks it against the .sha256 file beside it, and runs it with java -jar, in the gates of Java
// and Kotlin alone.
type Maven string

// Package returns the coordinates of the artifact of m without the version, as group:artifact.
func (m Maven) Package() string {
	pkg, _ := split(string(m))
	return pkg
}

// Version returns the version of m, the text after its last @.
func (m Maven) Version() string {
	_, v := split(string(m))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for an m that is not
// <group>:<artifact>@<version>.
func (m Maven) Validate() error {
	return validate("<group>:<artifact>@<version>", string(m), mavenName, version)
}

// Composer is a Composer package and its version, as <vendor>/<package>@<version>, such as
// phpstan/phpstan@2.2.17. ergon tool run installs the Composer packages of a section together into
// one project, so a package such as phpstan/phpstan-strict-rules extends another, in the gate of
// PHP alone.
type Composer string

// Package returns the name of the package of c, as vendor/package.
func (c Composer) Package() string {
	pkg, _ := split(string(c))
	return pkg
}

// Version returns the version of c, the text after its last @.
func (c Composer) Version() string {
	_, v := split(string(c))
	return v
}

// Validate returns an error that wraps [ErrInvalid] for a c that is not
// <vendor>/<package>@<version>.
func (c Composer) Validate() error {
	return validate("<vendor>/<package>@<version>", string(c), composerName, version)
}

// split returns the package and the version of tool: the text before and after its last @, and
// the whole text with an empty version for a tool without an @.
func split(tool string) (pkg, v string) {
	pkg, v, _ = strings.CutLast(tool, "@")
	return pkg, v
}

// validate returns an error that wraps [ErrInvalid] for a tool whose package does not match pkg
// or whose version does not match ver. form is the form of the kind, such as
// <package>@<version>, which the error states.
func validate(form, tool string, pkg, ver *regexp.Regexp) error {
	p, v := split(tool)
	if !pkg.MatchString(p) || !ver.MatchString(v) {
		return fmt.Errorf("%w: %q, which is not %s", ErrInvalid, tool, form)
	}
	return nil
}
