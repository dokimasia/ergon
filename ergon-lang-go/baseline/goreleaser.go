// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
)

// configFile is the name of the configuration of GoReleaser of a module, in the directory of the
// module.
const configFile = ".goreleaser.yaml"

// The template of the configuration of GoReleaser: its path in [goreleaserTemplate], and its
// delimiters, which leave the {{ }} of the templates of GoReleaser as text.
const (
	goreleaserName = "goreleaser/goreleaser.yaml.tmpl"
	leftDelim      = "{{%"
	rightDelim     = "%}}"
)

// The suffixes of the targets of GoReleaser for an architecture of Go: the level of amd64 and the
// version of arm64 that Go builds by default.
var targetSuffixes = map[string]string{amd64: "_v1", arm64: "_v8.0"}

// packedSuffix ends the identifier of the build of the Linux targets of a command, whose binaries
// UPX packs.
const packedSuffix = "-upx"

// goreleaserTemplate is the template of the configuration of GoReleaser of a module.
//
//go:embed goreleaser
var goreleaserTemplate embed.FS

// goreleaserConfig parses the template of the configuration of GoReleaser once, with the functions
// quote, which writes a string as a scalar of YAML in double quotes, and join, which writes a list
// of strings with a separator.
var goreleaserConfig = template.Must(template.New(path.Base(goreleaserName)).Delims(leftDelim, rightDelim).Funcs(
	template.FuncMap{"quote": strconv.Quote, "join": strings.Join},
).Option("missingkey=error").ParseFS(goreleaserTemplate, goreleaserName))

// release is the configuration of GoReleaser of one module, as its template reads it.
type release struct {
	// Path is the path of the configuration in the repository.
	Path string

	// Module is the directory of the module, . for the root module.
	Module string

	// Project is the project of GoReleaser: the name of the repository for the root module, and the
	// name of the repository with the directory of the module, its slashes as '-', for any other.
	Project string

	// Prefix is the prefix of the tags of the module: empty for the root module, and the directory of
	// the module with a slash for any other.
	Prefix string

	// Homepage is the address of the repository on GitHub.
	Homepage string

	// Vendor is the owner of the repository.
	Vendor string

	// Maintainer is the owner with the security contact, as Name <address>.
	Maintainer string

	// TapOwner and TapName are the owner and the name of the tap of the casks.
	TapOwner, TapName string

	// Builds are the builds of the commands of the module, in the order of binaries.
	Builds []build

	// Commands are the commands of the module, in the order of binaries.
	Commands []command

	// Completions reports that a command of the module writes completions.
	Completions bool

	// Packages reports that a command of the module has Linux packages.
	Packages bool

	// Homebrew reports that a command of the module has a cask.
	Homebrew bool

	// Notice reports that the repository has a NOTICE, which ergon init writes for Apache-2.0
	// alone. Each archive and each package of the release includes it.
	Notice bool
}

// build is a build of GoReleaser of a command, as its template reads it.
type build struct {
	// ID is the identifier of the build: the name of the command with -upx for its Linux targets,
	// and the name alone for its other targets.
	ID string

	// Name is the name of the binary.
	Name string

	// Main is the package of the command, relative to its module.
	Main string

	// Targets are the targets of GoReleaser of the build, such as linux_amd64_v1.
	Targets []string

	// Pack reports that UPX packs each binary of the build, which is true for the Linux targets.
	Pack bool
}

// command is a command of a configuration of GoReleaser, as its template reads it.
type command struct {
	// Name is the name of the binary.
	Name string

	// Main is the package of the command, relative to its module, which writes its completions.
	Main string

	// Description is the line of the packages and the cask.
	Description string

	// License is the SPDX identifier of the command.
	License string

	// IDs are the identifiers of the builds of the command, which its archives and packages take.
	IDs []string

	// Packages are the Linux packages of the command, such as deb.
	Packages []string

	// Completions reports that the command writes completions.
	Completions bool

	// Homebrew reports that the command has a cask.
	Homebrew bool
}

// Files returns the configuration of GoReleaser of each module that a command of o names, at
// <module>/.goreleaser.yaml and at .goreleaser.yaml for the root module, in the order of the first
// command of each module, and no file for options without a command. It takes the options at the
// baseline, which list no command, when o is not the section go. Each configuration builds,
// archives, packs and signs the commands of its module, as the package documentation states, with
// the owner, the security contact, the license and the repository of a. The archives and the
// packages of a repository under Apache-2.0 include its NOTICE. Files returns an error for a
// template that does not execute, which is a defect of the producer.
func (Producer) Files(a *language.Answers, o language.Options, _ *workflow.Contribution) ([]language.File, error) {
	opts, ok := o.(*Options)
	if !ok {
		return nil, nil
	}
	homepage := "https://github.com/" + string(a.Repository)
	owner, name, _ := strings.Cut(opts.Homebrew.Tap, "/")
	var releases []*release
	for i := range opts.Binaries {
		c := &opts.Binaries[i]
		k := slices.IndexFunc(releases, func(r *release) bool { return r.Module == c.Module })
		if k < 0 {
			r := &release{
				Path:       configFile,
				Module:     c.Module,
				Project:    a.Name,
				Homepage:   homepage,
				Vendor:     a.Owner,
				Maintainer: a.Owner + " <" + a.SecurityContact + ">",
				TapOwner:   owner,
				TapName:    name,
				Notice:     a.License == spdx.Apache20,
			}
			if c.Module != root {
				r.Path = path.Join(c.Module, configFile)
				r.Project += "-" + strings.ReplaceAll(c.Module, "/", "-")
				r.Prefix = c.Module + "/"
			}
			releases = append(releases, r)
			k = len(releases) - 1
		}
		r := releases[k]
		license := string(a.License)
		if c.License != "" {
			license = string(c.License)
		}
		unpacked := build{ID: c.Name, Name: c.Name, Main: c.Main}
		packed := build{ID: c.Name + packedSuffix, Name: c.Name, Main: c.Main, Pack: true}
		for _, p := range c.targets() {
			target := p.OS() + "_" + p.Arch() + targetSuffixes[p.Arch()]
			if p.OS() == linux {
				packed.Targets = append(packed.Targets, target)
			} else {
				unpacked.Targets = append(unpacked.Targets, target)
			}
		}
		var ids []string
		if len(unpacked.Targets) > 0 {
			r.Builds = append(r.Builds, unpacked)
			ids = append(ids, unpacked.ID)
		}
		if len(packed.Targets) > 0 {
			r.Builds = append(r.Builds, packed)
			ids = append(ids, packed.ID)
		}
		packages := make([]string, 0, len(c.Packages))
		for _, p := range c.Packages {
			packages = append(packages, string(p))
		}
		r.Commands = append(r.Commands, command{
			Name:        c.Name,
			Main:        c.Main,
			Description: c.Description,
			License:     license,
			IDs:         ids,
			Packages:    packages,
			Completions: c.Completions,
			Homebrew:    c.Homebrew,
		})
		r.Completions = r.Completions || c.Completions
		r.Packages = r.Packages || len(c.Packages) > 0
		r.Homebrew = r.Homebrew || c.Homebrew
	}
	files := make([]language.File, 0, len(releases))
	for _, r := range releases {
		var b bytes.Buffer
		if err := goreleaserConfig.Execute(&b, r); err != nil {
			return nil, fmt.Errorf("baseline: the configuration of GoReleaser of %s: %w", r.Module, err)
		}
		files = append(files, language.File{Path: r.Path, Content: b.Bytes()})
	}
	return files, nil
}
