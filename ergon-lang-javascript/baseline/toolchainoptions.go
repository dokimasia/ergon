// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// manifest is the file that pins the Node.js of a repository and its packages.
const manifest = "package.json"

// matrixVersion is the version of Node.js of the matrix of a job, as a workflow writes it.
const matrixVersion = "${{ matrix.version }}"

// present is the condition of a step of the js toolchain in a job of every repository: the
// repository has package.json.
const present = "hashFiles('" + manifest + "') != ''"

// ToolchainOptions are the options of the section js of .ergon.yaml. They set the package manager
// and the toolkit, the tools, the paths and the scan that JavaScript and TypeScript share, and the
// setup of their jobs.
type ToolchainOptions struct {
	// PackageManager is the package manager of the packages of package.json. It also runs their test
	// script and their scan, and ergon tool run starts the npm packages with it.
	PackageManager option.PackageManager `yaml:"package-manager" doc:"The package manager of the packages of package.json: npm, which installs package-lock.json, or pnpm, which installs pnpm-lock.yaml. It runs the test script and the scan of the packages, and ergon tool run starts the npm packages of every section with npx or with pnpm dlx."`

	// Toolkit is the toolkit that formats, lints and tests the sources.
	Toolkit Toolkit `yaml:"toolkit" doc:"The toolkit of fmt-js, lint-js and the tests: biome or vite-plus. biome formats and lints with the managed biome.json, checks the types of TypeScript with tsc and runs the test script. vite-plus runs vp fmt, vp check and vp test --run of the packages of the repository with the configuration of vite.config.ts, and ergon init renders .prettierignore in place of biome.json."`

	// Tools are the tools of the shared targets and of a release.
	Tools ToolchainTools `yaml:"tools" doc:"The tools of the targets that JavaScript and TypeScript share and of a release, as <package>@<version> of npm, which ergon tool run starts with the package manager."`

	// Paths are the paths that Biome formats and checks.
	Paths option.Paths `yaml:"paths" doc:"The paths that Biome formats and checks. vp reads its paths from vite.config.ts."`

	// Exclude are the globs of the paths that fmt-js and lint-js skip.
	Exclude Exclude `yaml:"exclude" doc:"The globs of the paths that fmt-js and lint-js skip. Each glob is relative to the root of the repository. The glob vendor matches the directory vendor, and src/**/*.gen.ts matches the files below src that end in .gen.ts. A glob has * inside a segment and ** only as a whole segment. biome skips the paths through files.includes of biome.json. vite-plus skips them through .prettierignore in vp fmt and in the format check of vp check, and through .eslintignore in the lint of vp check."`

	// Audit are the options of audit-js.
	Audit option.Threshold `yaml:"audit" doc:"audit-js fails on a known vulnerability of a package of the lockfile of severity or above: low, moderate, high or critical."`

	// CI are the pins of the setup of Node.js, the runners, the versions and the limit of the jobs of
	// JavaScript and TypeScript.
	CI option.MatrixCI[ToolchainActions] `yaml:"ci" doc:"The pins of setup-node and pnpm/setup, the runners and the versions of Node.js of the jobs check-javascript and check-typescript, and their limit in minutes. Empty runners select every runner of the section github, and empty versions the Node.js of package.json."`
}

var _ option.PackageManaged = (*ToolchainOptions)(nil)

// ToolchainTools are the tools of the targets that JavaScript and TypeScript share, and of a
// release.
type ToolchainTools struct {
	// Biome lints and formats the sources.
	Biome option.NPM `yaml:"biome" doc:"The linter and the formatter of lint-js and fmt-js, with the rules of biome.json, for the toolkit biome."`

	// NPM uploads the tarballs of a release.
	NPM option.NPM `yaml:"npm" doc:"The npm that uploads the tarballs of a release to the npm registry in the job publish of release.yml. The trusted publishing of npm needs npm 11.5.1 or later."`
}

// ToolchainActions are the pins of the actions of the setup of the js toolchain.
type ToolchainActions struct {
	// SetupNode installs Node.js and npm.
	SetupNode workflow.Action `yaml:"setup-node" doc:"The action that installs the Node.js of package.json or of the matrix, with npm, for the package manager npm."`

	// SetupPNPM is the action that installs pnpm, Node.js and the packages.
	SetupPNPM workflow.Action `yaml:"setup-pnpm" doc:"The action that installs pnpm, the Node.js of devEngines.runtime of package.json or of the matrix, and the packages of pnpm-lock.yaml, for the package manager pnpm."`
}

// Validate returns nil. The section js has no rule beyond the Validate methods of the types of its
// options, which the command calls before Validate.
func (*ToolchainOptions) Validate() error {
	return nil
}

// Manager returns the package manager of o, with which ergon tool run starts the npm packages of
// every section.
func (o *ToolchainOptions) Manager() option.PackageManager {
	return o.PackageManager
}

// Contribution returns the part of the js toolchain of the workflows for o:
//
//   - The setup that the jobs check-javascript and check-typescript run on the runners of o, once
//     the repository has package.json. For npm, setup-node installs the Node.js of package.json,
//     or the version of the matrix where o lists versions, and npm ci installs the packages. For
//     pnpm, pnpm/setup installs pnpm, the Node.js of devEngines.runtime or of the matrix, and the
//     packages of pnpm-lock.yaml, without which it fails. The steps of ci.steps run after the
//     install.
//   - The same install, with the Node.js of package.json, sets up the job version of version.yml and
//     the jobs pack and publish of release.yml in a repository with package.json. A release
//     refreshes the lockfile, packs the tarballs with the scripts of the packages and runs npm.
//   - The CodeQL analysis of javascript-typescript reads the sources without a build.
//   - Dependabot updates the npm packages, whose ecosystem npm includes pnpm.
//   - The job publish of release.yml uploads the tarballs to the npm registry, whose trusted
//     publishing exchanges the OIDC token of the job.
func (o *ToolchainOptions) Contribution() workflow.Contribution {
	release := o.install(nil)
	for i := range release {
		release[i].If = present
	}
	return workflow.Contribution{
		Setup: &workflow.Setup{
			Files:    manifest,
			Runners:  o.CI.Runners,
			Versions: o.CI.Versions,
			Steps:    slices.Concat(o.install(o.CI.Versions), o.CI.Steps),
			Timeout:  o.CI.Timeout,
		},
		Release: release,
		CodeQL: []workflow.CodeQL{{
			Language:  "javascript-typescript",
			Name:      "JavaScript and TypeScript",
			BuildMode: "none",
			Files:     manifest,
			Timeout:   o.CI.Timeout,
		}},
		Updates:  []workflow.Update{{Ecosystem: "npm", Directories: []string{"/"}}},
		Registry: true,
	}
}

// install returns the steps that install Node.js and the packages of o, as
// [ToolchainOptions.Contribution] states them for its package manager, with the Node.js of the
// matrix for versions and with the Node.js of package.json for none.
func (o *ToolchainOptions) install(versions []string) []workflow.Step {
	if o.PackageManager == option.PackageManagerPNPM {
		with := map[string]string{"require-lockfile": "true"}
		if len(versions) > 0 {
			with["runtime"] = "node@" + matrixVersion
		}
		return []workflow.Step{{
			Name: "Set up pnpm, Node.js and the packages",
			Uses: o.CI.Actions.SetupPNPM,
			With: with,
		}}
	}
	with := map[string]string{"node-version-file": manifest}
	if len(versions) > 0 {
		with = map[string]string{"node-version": matrixVersion}
	}
	return []workflow.Step{
		{Name: "Set up Node.js", Uses: o.CI.Actions.SetupNode, With: with},
		{Name: "Install the packages", Run: []string{"npm ci"}},
	}
}
