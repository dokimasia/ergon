// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
)

// The rules of the names of a command.
var (
	// commandName matches the name of a command: a lowercase letter or a digit, then lowercase
	// letters, digits and '-', which a binary, a package of Linux and a cask of Homebrew all accept.
	commandName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

	// commandPath matches the directory of a module and the package of a command: letters, digits,
	// '.', '_', '-' and '/', which a command of the shell takes without quotes.
	commandPath = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

// root is the directory of the root module.
const root = "."

// mainPrefix starts the package of a command, which is relative to its module.
const mainPrefix = "./"

// Package is a Linux package format of a command.
type Package string

// The package formats of a command.
const (
	// PackageDeb is a package of Debian and Ubuntu.
	PackageDeb Package = "deb"

	// PackageRPM is a package of Fedora, RHEL and openSUSE.
	PackageRPM Package = "rpm"

	// PackageAPK is a package of Alpine.
	PackageAPK Package = "apk"
)

// packages are the package formats of a command.
var packages = []Package{PackageDeb, PackageRPM, PackageAPK}

// Command is a command whose binaries each release of its module attaches: an archive for each of
// its platforms, its Linux packages, and its cask of Homebrew.
type Command struct {
	// Name is the name of the binary, and of its archives, packages and cask.
	Name string `yaml:"name"`

	// Module is the directory of the module in go.work, relative and slash-separated, or . for the
	// root module.
	Module string `yaml:"module"`

	// Main is the package of the command, relative to the module, such as ./cmd/ergon.
	Main string `yaml:"main"`

	// Description is one line that the packages and the cask state.
	Description string `yaml:"description"`

	// License is the SPDX identifier of the command, or empty for the license of the repository.
	License spdx.ID `yaml:"license"`

	// Platforms are the platforms of the builds, and every platform of a release binary when
	// empty.
	Platforms []option.Platform `yaml:"platforms"`

	// Packages are the Linux packages of the command.
	Packages []Package `yaml:"packages"`

	// Completions reports that the command is a cobra program, whose command completion writes the
	// completions of bash, zsh and fish.
	Completions bool `yaml:"completions"`

	// Homebrew reports that the tap of the section receives a cask of the command.
	Homebrew bool `yaml:"homebrew"`
}

// Validate returns an error that wraps [option.ErrInvalid] for the first value of c that a release
// cannot build:
//
//   - a Name outside lowercase letters, digits and '-'
//   - a Module that is not relative, clean and slash-separated, or has a character other than a
//     letter, a digit, '.', '_', '-' and '/'
//   - a Main that does not start with ./, is not clean, or has such a character
//   - a Description that is empty or spans lines
//   - a License that is not an identifier of --license
//   - a platform or a package that is not valid or is named twice
//   - a package without a Linux platform, and Homebrew without a darwin or Linux platform
func (c *Command) Validate() error {
	if !commandName.MatchString(c.Name) {
		return fmt.Errorf("%w: the command %q, whose name is not lowercase letters, digits and -", option.ErrInvalid,
			c.Name)
	}
	cleanModule := c.Module == root || (path.Clean(c.Module) == c.Module && !path.IsAbs(c.Module) &&
		!strings.HasPrefix(c.Module, "..") && commandPath.MatchString(c.Module))
	if !cleanModule {
		return fmt.Errorf("%w: the command %s has the module %q, which is no clean relative directory",
			option.ErrInvalid, c.Name, c.Module)
	}
	rest, ok := strings.CutPrefix(c.Main, mainPrefix)
	cleanMain := ok && rest != "" && path.Clean(rest) == rest && !strings.HasPrefix(rest, "..") &&
		commandPath.MatchString(rest)
	if !cleanMain {
		return fmt.Errorf("%w: the command %s has the package %q, which is no clean package under ./",
			option.ErrInvalid, c.Name, c.Main)
	}
	if strings.TrimSpace(c.Description) == "" || strings.ContainsAny(c.Description, "\r\n") {
		return fmt.Errorf("%w: the command %s has a description that is empty or spans lines", option.ErrInvalid,
			c.Name)
	}
	if c.License != "" && !c.License.Valid() {
		return fmt.Errorf("%w: the command %s has the license %q, which is no identifier of --license",
			option.ErrInvalid, c.Name, c.License)
	}
	for i, p := range c.Platforms {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("the command %s: %w", c.Name, err)
		}
		if slices.Contains(c.Platforms[:i], p) {
			return fmt.Errorf("%w: the command %s names the platform %s twice", option.ErrInvalid, c.Name, p)
		}
	}
	for i, p := range c.Packages {
		if !slices.Contains(packages, p) || slices.Contains(c.Packages[:i], p) {
			return fmt.Errorf("%w: the command %s has the package %q, which is not deb, rpm or apk, or is named twice",
				option.ErrInvalid, c.Name, p)
		}
	}
	if len(c.Packages) > 0 && !c.builds(linux) {
		return fmt.Errorf("%w: the command %s has packages of Linux and no Linux platform", option.ErrInvalid, c.Name)
	}
	if c.Homebrew && !c.builds(darwin) && !c.builds(linux) {
		return fmt.Errorf("%w: the command %s has a cask and no darwin or Linux platform", option.ErrInvalid, c.Name)
	}
	return nil
}

// targets returns the platforms of c, every platform of a release binary for c without platforms.
func (c *Command) targets() []option.Platform {
	if len(c.Platforms) == 0 {
		return []option.Platform{
			option.LinuxAMD64, option.LinuxARM64, option.DarwinAMD64, option.DarwinARM64, option.WindowsAMD64,
			option.WindowsARM64,
		}
	}
	return c.Platforms
}

// builds reports whether c builds for a platform of the system system, such as linux.
func (c *Command) builds(system string) bool {
	return slices.ContainsFunc(c.targets(), func(p option.Platform) bool { return p.OS() == system })
}
