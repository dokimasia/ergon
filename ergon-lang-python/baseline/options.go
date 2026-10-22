// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"regexp"
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// project is the file that pins the Python of a repository and its dependencies.
const project = "pyproject.toml"

// steps are the steps of the gate of Python, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// pypiName matches the name of a PyPI package, as PEP 508 defines it: letters and digits, with dots,
// hyphens and underscores between them.
var pypiName = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// Options are the options of the section python of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Python.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Python: the release binary of uv, and the packages of PyPI that ergon tool run runs through it, as <package>@<version>."`

	// Paths are the paths that ruff and mypy check.
	Paths option.Paths `yaml:"paths" doc:"The paths that ruff and mypy check."`

	// Check are the steps of check-python.
	Check option.Check `yaml:"check" doc:"The steps that check-python runs, in order: lint, test, generate or audit."`

	// Test are the options of test-python.
	Test option.Run `yaml:"test" doc:"test-python runs pytest with args."`

	// Generate are the options of generate-python and verify-generate-python.
	Generate option.Generate `yaml:"generate" doc:"generate-python runs command with args at the root of the repository. verify-generate-python, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command Python has neither target."`

	// Audit are the options of audit-python.
	Audit Audit `yaml:"audit" doc:"audit-python scans the packages of uv.lock with pip-audit, and accepts each advisory of ignore, by its identifier, such as PYSEC-2024-1."`

	// CI are the runners, the versions of Python and the limit of the job check-python.
	CI option.MatrixCI[struct{}] `yaml:"ci" doc:"The runners and the versions of Python of the job check-python, and its limit in minutes. uv installs Python, so the job runs no action of its own. Empty runners select every runner of the section github, and empty versions the version of pyproject.toml."`
}

// Tools are the tools of the targets of Python.
type Tools struct {
	// UV runs the packages of PyPI, and installs Python.
	UV option.UV `yaml:"uv" doc:"The release of uv, which installs Python and the environment of the project, and runs the packages of PyPI of the section."`

	// Ruff lints and formats the sources.
	Ruff option.PyPI `yaml:"ruff" doc:"The linter and the formatter of lint-python and fmt-python, with the rules of ruff.toml."`

	// Mypy checks the types of the sources.
	Mypy option.PyPI `yaml:"mypy" run:"project" doc:"The type checker of lint-python, in its strictest mode, in the environment of the project."`

	// Pytest runs the tests.
	Pytest option.PyPI `yaml:"pytest" run:"project" doc:"The test runner of test-python, in the environment of the project."`

	// PipAudit scans the packages for known vulnerabilities.
	PipAudit option.PyPI `yaml:"pip-audit" doc:"The vulnerability scan of audit-python, over the packages of uv.lock."`
}

// Audit are the options of audit-python: the advisories that the scan accepts, and the packages of
// uv.lock that it leaves out.
type Audit struct {
	option.Audit `yaml:",inline"`

	// Skip are the packages of uv.lock without a release on PyPI, such as a package from a git
	// repository or from another index. pip-audit audits a package through its release on PyPI, and
	// fails for a package without one.
	Skip []string `yaml:"skip" doc:"The packages of uv.lock without a release on PyPI, such as a package from a git repository or from another index. audit-python leaves them out of the scan, because pip-audit fails for them."`
}

// Validate returns the error of [option.Audit.Validate] for the advisories of a. It returns an
// error that wraps [option.ErrInvalid] for a package of Skip that is not the name of a PyPI
// package, and for a package that Skip names twice.
func (a Audit) Validate() error {
	if err := a.Audit.Validate(); err != nil {
		return err
	}
	for i, pkg := range a.Skip {
		if !pypiName.MatchString(pkg) || slices.Contains(a.Skip[:i], pkg) {
			return fmt.Errorf("%w: skip %q, which is not the name of a PyPI package or is named twice",
				option.ErrInvalid, pkg)
		}
	}
	return nil
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Python does
// not have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of Python of the workflows for o:
//
//   - The job check-python runs make check-python on the runners of o, once the repository has
//     pyproject.toml. uv installs the Python of pyproject.toml, or the version of the matrix through
//     UV_PYTHON where o lists versions. The steps of ci.steps run before the installation of
//     ergon. The job keeps uv, which ergon tool run installs, in the cache of GitHub Actions.
//   - The CodeQL analysis of python reads the sources without a build.
//   - Dependabot updates uv.lock.
//   - The job publish of release.yml uploads the distributions to PyPI, whose trusted publishing
//     exchanges the OIDC token of the job.
func (o *Options) Contribution() workflow.Contribution {
	var env map[string]string
	if len(o.CI.Versions) > 0 {
		env = map[string]string{"UV_PYTHON": "${{ matrix.version }}"}
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-python",
			Name:        "Python",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    project,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Env:      env,
				Steps:    o.CI.Steps,
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check Python", Run: []string{"make check-python"}}},
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "python", Name: "Python", BuildMode: "none", Files: project, Timeout: o.CI.Timeout},
		},
		Updates:  []workflow.Update{{Ecosystem: "uv", Directories: []string{"/"}}},
		Registry: true,
	}
}
