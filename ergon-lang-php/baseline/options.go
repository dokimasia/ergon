// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// pin is the file that pins the PHP of a repository.
const pin = ".php-version"

// steps are the steps of the gate of PHP, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepAudit}

// Options are the options of the section php of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of PHP.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of PHP, as <vendor>/<package>@<version> of Packagist, which ergon tool run installs together into one Composer project of its own. Each tool accepts another version of its package and no other package."`

	// Paths are the paths that PHP-CS-Fixer and PHPStan check.
	Paths option.Paths `yaml:"paths" doc:"The paths that PHP-CS-Fixer and PHPStan check."`

	// Check are the steps of check-php.
	Check option.Check `yaml:"check" doc:"The steps that check-php runs, in order: lint, test or audit."`

	// Test are the options of test-php.
	Test option.Run `yaml:"test" doc:"test-php runs PHPUnit with args."`

	// CI are the pin of the setup of PHP, the runners, the versions and the limit of the job
	// check-php.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-php, the runners and the versions of PHP of the job check-php, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the PHP of .php-version."`
}

// Tools are the tools of the targets of PHP.
type Tools struct {
	// PHPStan analyzes the sources.
	PHPStan option.Composer `yaml:"phpstan" doc:"The static analyzer of lint-php, at its strictest level, with the rules of phpstan.dist.neon."`

	// StrictRules adds the strict rules to PHPStan.
	StrictRules option.Composer `yaml:"phpstan-strict-rules" doc:"The strict rules of PHPStan, which the extension installer loads."`

	// ExtensionInstaller loads the extensions of PHPStan that the project of the tools has.
	ExtensionInstaller option.Composer `yaml:"phpstan-extension-installer" doc:"The plugin of Composer that loads the extensions of PHPStan, such as phpstan-strict-rules."`

	// PHPCSFixer formats the sources.
	PHPCSFixer option.Composer `yaml:"php-cs-fixer" program:"php-cs-fixer" doc:"The formatter of fmt-php and lint-php, with the PER coding style."`
}

// Actions are the pins of the actions of the job check-php.
type Actions struct {
	// SetupPHP installs PHP and Composer.
	SetupPHP workflow.Action `yaml:"setup-php" doc:"The action that installs Composer and the PHP of .php-version or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a tool whose package is not the
// package of the baseline. The targets run the programs of PHPStan and PHP-CS-Fixer with the flags
// of those programs, and the extension installer of PHPStan loads phpstan-strict-rules, so a tool
// accepts another version alone. The command checks the format of each tool by the Validate method
// of its type before it calls Validate.
func (t *Tools) Validate() error {
	pins := []struct {
		key       string
		got, want option.Composer
	}{
		{key: "phpstan", got: t.PHPStan, want: tools.PHPStan},
		{key: "phpstan-strict-rules", got: t.StrictRules, want: tools.StrictRules},
		{key: "phpstan-extension-installer", got: t.ExtensionInstaller, want: tools.ExtensionInstaller},
		{key: "php-cs-fixer", got: t.PHPCSFixer, want: tools.PHPCSFixer},
	}
	for _, p := range pins {
		if p.got.Package() != p.want.Package() {
			return fmt.Errorf("%w: %s %q, which is not a version of %s", option.ErrInvalid, p.key, p.got,
				p.want.Package())
		}
	}
	return nil
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that PHP does not
// have. The command checks each option by the Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	return o.Check.Only(steps...)
}

// Contribution returns the part of PHP of the workflows for o:
//
//   - The job check-php runs make check-php on the runners of o, once the repository has
//     .php-version. setup-php installs Composer and the PHP of .php-version, or the version of the
//     matrix where o lists versions. The job keeps the Composer packages of its tools, which ergon
//     tool run installs, in the cache of GitHub Actions.
//   - Dependabot updates the Composer packages.
//
// CodeQL does not analyze PHP, so the contribution has no analysis.
func (o *Options) Contribution() workflow.Contribution {
	with := map[string]string{"php-version-file": pin}
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"php-version": "${{ matrix.version }}"}
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-php",
			Name:        "PHP",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    pin,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps:    []workflow.Step{{Name: "Set up PHP", Uses: o.CI.Actions.SetupPHP, With: with}},
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check PHP", Run: []string{"make check-php"}}},
		}},
		Updates: []workflow.Update{{Ecosystem: "composer", Directories: []string{"/"}}},
	}
}
