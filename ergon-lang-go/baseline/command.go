// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

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

// The rules of brew audit of Homebrew 7.0 for the description of a cask.
var (
	// caskArticle matches a description that starts with an article.
	caskArticle = regexp.MustCompile(`^(?i:the|an?)\s`)

	// caskCommandLine matches command line and commandline, which a description writes as
	// command-line.
	caskCommandLine = regexp.MustCompile(`(?i)command ?line`)

	// caskPlatform matches a name of macOS.
	caskPlatform = regexp.MustCompile(`(?i)\b(?:macOS|Mac(?: ?OS(?: ?X)?)?|OS ?X)\b`)

	// caskVirtualMachines matches the words after a name of macOS that brew audit accepts.
	caskVirtualMachines = regexp.MustCompile(`(?i)^ virtual machines?`)

	// caskSymbol matches an emoji or another character of the Unicode category So.
	caskSymbol = regexp.MustCompile(`\p{So}`)
)

// caskLowercaseWords are the words that the description of a cask may start with in lowercase.
var caskLowercaseWords = []string{"iOS", "iPhone", "macOS"}

// The limits of the description of a cask.
const (
	// caskDescriptionLength is the most characters that the description of a cask has.
	caskDescriptionLength = 80

	// caskWhiteSpace are the characters that the description of a cask neither starts nor ends with.
	caskWhiteSpace = " \t\r\n\f\v"

	// caskEtc is the one ending with a full stop that the description of a cask may have.
	caskEtc = "etc."

	// caskMAC is the name of the platform that brew audit leaves to a MAC address.
	caskMAC = "MAC"
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

	// Description is one line that the packages and the cask state. A command with a cask has a
	// description that brew audit accepts in a cask.
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
//   - with Homebrew, a Description that brew audit refuses in a cask, as caskDescriptionProblem
//     states the rules
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
	if c.Homebrew {
		if problem := caskDescriptionProblem(c.Name, c.Description); problem != "" {
			return fmt.Errorf("%w: the command %s has a description that brew audit refuses in a cask, because %s",
				option.ErrInvalid, c.Name, problem)
		}
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

// caskDescriptionProblem returns why brew audit refuses description as the description of the cask
// name, and the empty string for a description that it accepts. It applies the rules of brew audit
// in their order, and returns the first that description breaks: no white space at its start or its
// end, command-line with its hyphen, no article, no lowercase letter and no name of the cask at its
// start, no name of macOS, no full stop at its end unless it ends with etc., no emoji or other
// symbol, and at most 80 characters. It accepts a name of macOS that " virtual machine" or
// " virtual machines" follows, and the word MAC, as brew audit does. name is the name of a command,
// and description is not empty, as Validate checks before.
func caskDescriptionProblem(name, description string) string {
	if strings.Trim(description, caskWhiteSpace) != description {
		return "it starts or ends with white space"
	}
	if caskCommandLine.MatchString(description) {
		return "it writes command line without the hyphen of command-line"
	}
	if caskArticle.MatchString(description) {
		return "it starts with an article"
	}
	first := strings.Fields(description)[0]
	if description[0] >= 'a' && description[0] <= 'z' && !slices.Contains(caskLowercaseWords, first) {
		return "it starts with a lowercase letter"
	}
	letters := strings.Split(strings.ReplaceAll(name, "-", ""), "")
	for i, l := range letters {
		letters[i] = regexp.QuoteMeta(l)
	}
	if regexp.MustCompile(`(?i)^` + strings.Join(letters, `[\s\-]?`) + `\b`).MatchString(description) {
		return "it starts with the name of the command"
	}
	for _, m := range caskPlatform.FindAllStringIndex(description, -1) {
		if caskVirtualMachines.MatchString(description[m[1]:]) {
			continue
		}
		if description[m[0]:m[1]] != caskMAC {
			return "it names macOS"
		}
		break
	}
	if strings.HasSuffix(description, ".") && !strings.HasSuffix(description, caskEtc) {
		return "it ends with a full stop"
	}
	if caskSymbol.MatchString(description) {
		return "it contains an emoji or another symbol"
	}
	if utf8.RuneCountInString(description) > caskDescriptionLength {
		return fmt.Sprintf("it has more than %d characters", caskDescriptionLength)
	}
	return ""
}
