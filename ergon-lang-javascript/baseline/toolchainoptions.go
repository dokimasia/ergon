// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// manifest is the file that pins the Node.js of a repository and its packages.
const manifest = "package.json"

// ToolchainOptions are the options of the section js of .ergon.yaml: the tools, the paths and the
// scan that JavaScript and TypeScript share, and the setup of their jobs.
type ToolchainOptions struct {
	// Tools are the tools of the shared targets.
	Tools ToolchainTools `yaml:"tools" doc:"The tools of the targets that JavaScript and TypeScript share, as <package>@<version> of npm, which ergon tool run runs with npx."`

	// Paths are the paths that Biome formats and checks.
	Paths option.Paths `yaml:"paths" doc:"The paths that Biome formats and checks."`

	// Audit are the options of audit-js.
	Audit option.Threshold `yaml:"audit" doc:"audit-js fails on a known vulnerability of a package of package-lock.json of severity or above: low, moderate, high or critical."`

	// CI are the pin of the setup of Node.js, the runners, the versions and the limit of the jobs of
	// JavaScript and TypeScript.
	CI option.MatrixCI[ToolchainActions] `yaml:"ci" doc:"The pin of setup-node, the runners and the versions of Node.js of the jobs check-javascript and check-typescript, and their limit in minutes. Empty runners select every runner of the section github, and empty versions the Node.js of package.json."`
}

// ToolchainTools are the tools of the targets that JavaScript and TypeScript share.
type ToolchainTools struct {
	// Biome lints and formats the sources.
	Biome option.NPM `yaml:"biome" doc:"The linter and the formatter of lint-js and fmt-js, with the rules of biome.json."`
}

// ToolchainActions are the pins of the actions of the setup of the js toolchain.
type ToolchainActions struct {
	// SetupNode installs Node.js and npm.
	SetupNode workflow.Action `yaml:"setup-node" doc:"The action that installs the Node.js of package.json or of the matrix, with npm."`
}

// Validate returns nil. The section js has no rule beyond the Validate methods of the types of its
// options, which the command calls before Validate.
func (*ToolchainOptions) Validate() error {
	return nil
}

// Contribution returns the part of the js toolchain of the workflows for o:
//
//   - The setup that the jobs check-javascript and check-typescript run on the runners of o, once
//     the repository has package.json: setup-node installs the Node.js of package.json, or the
//     version of the matrix where o lists versions, and npm ci installs the packages.
//   - The CodeQL analysis of javascript-typescript reads the sources without a build.
//   - Dependabot updates the npm packages.
func (o *ToolchainOptions) Contribution() workflow.Contribution {
	with := map[string]string{"node-version-file": manifest}
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"node-version": "${{ matrix.version }}"}
	}
	return workflow.Contribution{
		Setup: &workflow.Setup{
			Files:    manifest,
			Runners:  o.CI.Runners,
			Versions: o.CI.Versions,
			Steps: []workflow.Step{
				{Name: "Set up Node.js", Uses: o.CI.Actions.SetupNode, With: with},
				{Name: "Install the packages", Run: []string{"npm ci"}},
			},
			Timeout: o.CI.Timeout,
		},
		CodeQL: []workflow.CodeQL{{
			Language:  "javascript-typescript",
			Name:      "JavaScript and TypeScript",
			BuildMode: "none",
			Files:     manifest,
			Timeout:   o.CI.Timeout,
		}},
		Updates: []workflow.Update{{Ecosystem: "npm", Directories: []string{"/"}}},
	}
}
