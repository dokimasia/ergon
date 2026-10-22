// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// repository is the repository on GitHub whose releases publish spec and its reusable workflows.
const repository = "stealth-scale/spec"

// Role is what a repository is to spec. Each role is the name of the section of .spec.yaml that
// states it.
type Role string

// The roles of a repository.
const (
	// Definition is the repository of a conformance definition, whose .spec.yaml has the section
	// definition.
	Definition Role = "definition"

	// Implementation is a repository with copies of definitions, whose .spec.yaml has the section
	// implements.
	Implementation Role = "implementation"
)

// Valid reports whether r is [Definition] or [Implementation].
func (r Role) Valid() bool {
	return r == Definition || r == Implementation
}

// Spec is the release binary of spec, stealth-scale/spec on GitHub, whose release publishes a
// .tar.gz for every platform.
type Spec struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = Spec{}

// Asset returns the archive of spec for p from the release of the version of s on GitHub. The
// archive is spec_<version>_<os>_<arch>.tar.gz on every platform, with the program spec at its
// root, and spec.exe on Windows. It returns an error that wraps [option.ErrNoAsset] for a platform
// that is not valid.
func (s Spec) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: spec has no asset for %q", option.ErrNoAsset, p)
	}
	program := "spec"
	if p.OS() == windows {
		program += ".exe"
	}
	download := "https://github.com/" + repository + "/releases/download/v" + s.Version
	return option.Asset{
		URL:     download + "/spec_" + s.Version + "_" + p.OS() + "_" + p.Arch() + ".tar.gz",
		Program: program,
	}, nil
}

// Repository returns stealth-scale/spec, the repository on GitHub whose releases publish spec.
func (Spec) Repository() string {
	return repository
}

// Tools are the tools of the section spec.
type Tools struct {
	// Spec is the program spec.
	Spec Spec `yaml:"spec" doc:"The release of spec, which checks the copies of the definitions and the manifest of a definition."`
}

// Actions are the pins of the actions of the section spec.
type Actions struct {
	// Propose is the reusable workflow propose.yml of spec, which spec.yml calls.
	Propose workflow.Action `yaml:"propose" doc:"The reusable workflow propose.yml of spec, which spec.yml calls on each weekday."`
}

// Options are the options of the section spec of .ergon.yaml.
type Options struct {
	// Roles are what the repository is to spec, each once.
	Roles []Role `yaml:"roles" doc:"What the repository is to spec: definition for the repository of a conformance definition, and implementation for a repository with copies of definitions. Each role needs its section of .spec.yaml, definition or implements. A repository without a role gets no target, job or workflow of spec."`

	// Tools are the tools of spec.
	Tools Tools `yaml:"tools" doc:"The tools of spec."`

	// CI are the pin of the reusable workflow of spec and the limit of the job spec.
	CI option.CI[Actions] `yaml:"ci" doc:"The pin of the reusable workflow propose.yml of spec, which spec.yml calls, and the limit in minutes of the job spec of ci.yml."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a role that is not valid, as
// [Role.Valid] states, and for a role that Roles lists twice. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	for i, r := range o.Roles {
		if !r.Valid() {
			return fmt.Errorf("%w: the role %q, which is neither %s nor %s", option.ErrInvalid, r, Definition,
				Implementation)
		}
		if slices.Contains(o.Roles[:i], r) {
			return fmt.Errorf("%w: roles lists the role %s twice", option.ErrInvalid, r)
		}
	}
	return nil
}
