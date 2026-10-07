// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// The files that the job of Go reads: the workspace of the repository, which pins the version of
// Go, and the modules.
const (
	workspace = "go.work"
	modules   = "**/go.mod"
)

// steps are the steps of the gate of Go, which the key check of the section names.
var steps = []option.Step{
	option.StepLint, option.StepTest, option.StepRace, option.StepFuzz, option.StepBench, option.StepMutate,
	option.StepGenerate, option.StepAudit,
}

// Options are the options of the section go of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Go.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Go, which ergon tool run installs with go install, as <module>@<version>."`

	// Paths are the package patterns of the targets in every module.
	Paths option.Paths `yaml:"paths" doc:"The package patterns that each target of Go passes in every module of go.work."`

	// Check are the steps of check-go.
	Check option.Check `yaml:"check" doc:"The steps that check-go runs, in order: lint, test, race, fuzz, bench, mutate, generate or audit."`

	// Lint are the options of lint-go.
	Lint Lint `yaml:"lint" doc:"The options of lint-go, which runs golangci-lint with the rules of .golangci.yml, ergon-go-vet and go mod tidy -diff in every module."`

	// Test are the options of test-go.
	Test option.Run `yaml:"test" doc:"test-go runs go test with args."`

	// Race are the options of race-go.
	Race option.Run `yaml:"race" doc:"race-go runs go test -race with args. -p=1 tests one package at a time, because the race detector multiplies the memory of a test."`

	// Fuzz are the options of fuzz-go.
	Fuzz option.Fuzz `yaml:"fuzz" doc:"fuzz-go fuzzes each fuzz target whose name matches the regular expression match, for time each, with args. -fuzzminimizetime bounds the minimization of a new input, which go test runs for 60s by default."`

	// Bench are the options of bench-go.
	Bench option.Bench `yaml:"bench" doc:"bench-go runs each benchmark whose name matches the regular expression match, count times, for time each, with args. benchstat states a 95% confidence interval from 6 runs."`

	// Mutate are the options of mutate-go.
	Mutate option.Mutate `yaml:"mutate" doc:"mutate-go runs dokimi-mutate-go with workers mutants of a module at a time and args, and ends the run after timeout. A timeout of 0s sets no limit."`

	// Generate are the options of generate-go.
	Generate option.Run `yaml:"generate" doc:"generate-go runs go generate with args, and fails when the run changes a tracked file or writes a file that git does not track."`

	// Audit are the options of audit-go.
	Audit option.Run `yaml:"audit" doc:"audit-go runs govulncheck with args in every module."`

	// CI are the pins of the actions of the job check-go, its runners, its versions of Go and its
	// limit.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-go, the runners and the versions of Go of the job check-go, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the version of go.work."`
}

// Tools are the tools of the targets of Go.
type Tools struct {
	// GolangCILint lints and formats the sources.
	GolangCILint option.Module `yaml:"golangci-lint" doc:"The linter and the formatter of lint-go and fmt-go, with the rules of .golangci.yml."`

	// Govulncheck scans the modules for known vulnerabilities.
	Govulncheck option.Module `yaml:"govulncheck" doc:"The vulnerability scan of audit-go, which reports a known vulnerability that the code calls."`

	// Benchstat compares two runs of the benchmarks.
	Benchstat option.Module `yaml:"benchstat" doc:"The comparison of two runs of bench-go in benchstat-go."`

	// DokimiMutateGo runs the mutation tests.
	DokimiMutateGo option.Module `yaml:"dokimi-mutate-go" doc:"The mutation engine of mutate-go."`

	// ErgonGoVet runs the analyzers of ergon.
	ErgonGoVet option.Module `yaml:"ergon-go-vet" doc:"The analyzers of lint-go: errorprefix, which reports an error whose text does not start with the name of its package, and skipexpiry, which reports a skipped test whose date has passed."`
}

// Lint are the options of lint-go.
type Lint struct {
	// Exclude are the package patterns that ergon-go-vet skips.
	Exclude option.Paths `yaml:"exclude" doc:"The package patterns that ergon-go-vet skips, such as ./internal/legacy/...."`
}

// Actions are the pins of the actions of the job check-go.
type Actions struct {
	// SetupGo installs Go.
	SetupGo workflow.Action `yaml:"setup-go" doc:"The action that installs Go, at the version of go.work or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Go does not
// have. The command checks each option by the Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	return o.Check.Only(steps...)
}

// Contribution returns the part of Go of the workflows for o:
//
//   - The job check-go runs make check-go on the runners and the versions of Go of o, once the
//     repository has a go.mod. It fails a repository with a go.mod and without go.work, because the
//     targets run in the modules of go.work. setup-go installs the version of go.work, or the
//     version of the matrix where o lists versions, and caches the modules by every go.sum.
//   - The CodeQL analysis of go builds the modules with autobuild, once the repository has go.work.
//   - Dependabot updates the modules of every directory.
func (o *Options) Contribution() workflow.Contribution {
	with := map[string]string{"go-version-file": workspace, "cache-dependency-path": "**/go.sum"}
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"go-version": "${{ matrix.version }}", "cache-dependency-path": "**/go.sum"}
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-go",
			Name:        "Go",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    modules,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps: []workflow.Step{
					{
						Name: "Require go.work",
						If:   "hashFiles('" + workspace + "') == ''",
						Run: []string{
							`echo "::error::The targets of Go run in the modules of go.work, which the repository lacks."`,
							"exit 1",
						},
					},
					{Name: "Set up Go", Uses: o.CI.Actions.SetupGo, With: with},
				},
				Timeout: o.CI.Timeout,
			},
			Steps: []workflow.Step{{Name: "Check Go", Run: []string{"make check-go"}}},
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "go", Name: "Go", BuildMode: "autobuild", Files: workspace, Timeout: o.CI.Timeout},
		},
		Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/", "/**/*"}}},
	}
}
