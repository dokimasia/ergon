// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// The files that the job of Go reads: the workspace of the repository, which pins the version of
// Go, and the modules.
const (
	workspace = "go.work"
	modules   = "**/go.mod"
)

// setupName is the name of the step that installs Go in each job of Go.
const setupName = "Set up Go"

// steps are the steps of the gate of Go, which the key check of the section names.
var steps = []option.Step{
	option.StepLint, option.StepTest, option.StepRace, option.StepFuzz, option.StepBench, option.StepMutate,
	option.StepGenerate, option.StepAudit,
}

// nightly are the steps of Go that are too long for each push, which the key nightly names.
var nightly = []option.Step{option.StepFuzz, option.StepBench, option.StepMutate}

// tapRepository matches a repository on GitHub as owner/name.
var tapRepository = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// Options are the options of the section go of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets and of the releases of Go.
	Tools Tools `yaml:"tools" doc:"The tools of the targets and of the releases of Go, which ergon tool run installs: a Go module with go install, as <module>@<version>, and a release binary by its version and the SHA-256 of the asset of each platform."`

	// Paths are the package patterns of the targets in every module.
	Paths option.Paths `yaml:"paths" doc:"The package patterns that each target of Go passes in every module of go.work."`

	// Check are the steps of check-go.
	Check option.Check `yaml:"check" doc:"The steps that check-go runs, in order: lint, test, race, fuzz, bench, mutate, generate or audit."`

	// Nightly are the steps that nightly.yml runs, with the limit of each.
	Nightly option.Nightly `yaml:"nightly" doc:"The steps that nightly.yml runs on its schedule, fuzz, bench or mutate, each in a job of its own on Linux, mapped to the limit of that job in minutes."`

	// Lint are the options of lint-go.
	Lint Lint `yaml:"lint" doc:"The options of lint-go, which runs golangci-lint with the rules of .golangci.yml and ergon-go-vet in every module."`

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

	// Generate are the options of generate-go and verify-generate-go.
	Generate option.Generate `yaml:"generate" doc:"generate-go runs command with args and the paths in every module. verify-generate-go, the step generate of check, runs the same and fails when the run changes a tracked file or writes a file that git does not track. An empty command removes both targets."`

	// Audit are the options of audit-go.
	Audit option.Run `yaml:"audit" doc:"audit-go runs govulncheck with args in every module."`

	// Binaries are the commands whose binaries each release of their module attaches.
	Binaries []Command `yaml:"binaries" doc:"The commands whose binaries each release of their module attaches, which the job pack of release.yml builds, signs and attests with GoReleaser: name, the binary and its archives, packages and cask; module, the directory of the module in go.work, . for the root; main, the package of the command in the module; description, one line, and for a cask one that brew audit accepts, such as one without a full stop at its end; platforms, every platform when empty; completions, true for a cobra program; packages, from deb, rpm and apk; homebrew, true for a cask in homebrew.tap; license, the license of the repository when empty."`

	// Homebrew are the options of the casks of the commands.
	Homebrew Homebrew `yaml:"homebrew" doc:"The options of the casks of the commands under binaries whose homebrew is true."`

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

	// GoReleaser builds the binaries of the commands.
	GoReleaser GoReleaser `yaml:"goreleaser" doc:"The release of GoReleaser, which builds, packs and signs the commands of binaries in the job pack of release.yml."`

	// Cosign signs the checksums of a release.
	Cosign Cosign `yaml:"cosign" doc:"The release of cosign, which signs the checksums.txt of each release with the OIDC identity of the workflow, without a key."`

	// Syft writes the SBOMs of a release.
	Syft Syft `yaml:"syft" doc:"The release of syft, which writes the SPDX SBOM of each archive and package of a release."`

	// UPX packs the Linux binaries of a release.
	UPX UPX `yaml:"upx" doc:"The release of UPX, which packs each Linux binary of the commands of binaries with --best --lzma in the job pack of release.yml."`
}

// Lint are the options of lint-go.
type Lint struct {
	// Exclude are the package patterns that ergon-go-vet skips.
	Exclude option.Paths `yaml:"exclude" doc:"The package patterns that ergon-go-vet skips, such as ./internal/legacy/...."`
}

// Homebrew are the options of the casks of the commands.
type Homebrew struct {
	// Tap is the repository of the tap on GitHub, as owner/name, or empty for no tap.
	Tap string `yaml:"tap" doc:"The repository of the Homebrew tap on GitHub, as owner/name, to which the job homebrew of release.yml commits each cask, with a token of the GitHub App of ERGON_APP_CLIENT_ID and ERGON_APP_PRIVATE_KEY."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a Tap that is neither empty nor
// owner/name.
func (h Homebrew) Validate() error {
	if h.Tap != "" && !tapRepository.MatchString(h.Tap) {
		return fmt.Errorf("%w: homebrew.tap %q, which is not owner/name", option.ErrInvalid, h.Tap)
	}
	return nil
}

// Actions are the pins of the actions of the job check-go.
type Actions struct {
	// SetupGo installs Go.
	SetupGo workflow.Action `yaml:"setup-go" doc:"The action that installs Go, at the version of go.work or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Go does not
// have, for the step generate without a command, for a step of nightly other than fuzz, bench and
// mutate, for a command that is not valid, as [Command.Validate] states, for two commands of one
// name, for a command named after the UPX build of another, <name>-upx, and for a command with a
// cask while homebrew.tap is empty. The command checks each option by the Validate method of its
// type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	if err := o.Nightly.Only(nightly...); err != nil {
		return err
	}
	if err := o.Generate.CheckStep(o.Check); err != nil {
		return err
	}
	for i := range o.Binaries {
		c := &o.Binaries[i]
		if err := c.Validate(); err != nil {
			return err
		}
		if slices.ContainsFunc(o.Binaries[:i], func(d Command) bool { return d.Name == c.Name }) {
			return fmt.Errorf("%w: binaries names the command %s twice", option.ErrInvalid, c.Name)
		}
		base, packs := strings.CutSuffix(c.Name, packedSuffix)
		if packs && slices.ContainsFunc(o.Binaries, func(d Command) bool { return d.Name == base }) {
			return fmt.Errorf("%w: the command %s has the name of the UPX build of the command %s", option.ErrInvalid,
				c.Name, base)
		}
		if c.Homebrew && o.Homebrew.Tap == "" {
			return fmt.Errorf("%w: the command %s has a cask, and homebrew.tap is empty", option.ErrInvalid, c.Name)
		}
	}
	return nil
}

// Contribution returns the part of Go of the workflows for o:
//
//   - The job check-go runs make --keep-going check-go on the runners and the versions of Go of o,
//     once the repository has a go.mod, so a step that fails does not skip the steps after it. Each
//     step runs in every module and fails after the last. The job fails a repository with a go.mod
//     and without go.work, because the targets run in the modules of go.work. setup-go installs the
//     version of go.work, or the version of the matrix where o lists versions, and caches the
//     modules by every go.sum. The job keeps the tools of the section, which ergon tool run
//     installs, in the cache of GitHub Actions.
//   - The job <step>-go of nightly.yml runs make <step>-go for each step of nightly, in the order of
//     the targets, on the Linux runner of the section github and the version of go.work, with the
//     limit that nightly states for the step. It sets up Go as check-go does.
//   - The release steps install the version of go.work with setup-go in the jobs version and pack of
//     release.yml, once the repository has go.work, for the go mod tidy of a release and the builds
//     of its commands.
//   - The assets of the releases of the commands of binaries, with the tap of homebrew when a
//     command has a cask, and no assets without a command.
//   - The CodeQL analysis of go builds the modules with autobuild, once the repository has go.work.
//   - Dependabot updates the modules of every directory.
func (o *Options) Contribution() workflow.Contribution {
	pinned := map[string]string{"go-version-file": workspace, "cache-dependency-path": "**/go.sum"}
	with := pinned
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"go-version": "${{ matrix.version }}", "cache-dependency-path": "**/go.sum"}
	}
	require := workflow.Step{
		Name: "Require go.work",
		If:   "hashFiles('" + workspace + "') == ''",
		Run: []string{
			`echo "::error::The targets of Go run in the modules of go.work, which the repository lacks."`,
			"exit 1",
		},
	}
	read := map[string]string{"contents": "read"}
	nightly := o.Nightly.Steps()
	scheduled := make([]workflow.Job, 0, len(nightly))
	for _, s := range nightly {
		target := s.Target(Name)
		name := strings.ToUpper(string(s[:1])) + string(s[1:]) + " Go"
		scheduled = append(scheduled, workflow.Job{
			ID:          target,
			Name:        name,
			Permissions: read,
			Setup: &workflow.Setup{
				Files:   modules,
				Steps:   []workflow.Step{require, {Name: setupName, Uses: o.CI.Actions.SetupGo, With: pinned}},
				Timeout: o.Nightly[s],
			},
			Tools: true,
			Steps: []workflow.Step{{Name: name, Run: []string{"make " + target}}},
		})
	}
	var assets *workflow.Assets
	if len(o.Binaries) > 0 {
		assets = &workflow.Assets{}
		if slices.ContainsFunc(o.Binaries, func(c Command) bool { return c.Homebrew }) {
			assets.Tap = o.Homebrew.Tap
		}
	}
	return workflow.Contribution{
		Assets: assets,
		Jobs: []workflow.Job{{
			ID:          "check-go",
			Name:        "Go",
			Permissions: read,
			Setup: &workflow.Setup{
				Files:    modules,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps:    []workflow.Step{require, {Name: setupName, Uses: o.CI.Actions.SetupGo, With: with}},
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check Go", Run: []string{"make --keep-going check-go"}}},
		}},
		Nightly: scheduled,
		Release: []workflow.Step{{
			Name: setupName,
			If:   "hashFiles('" + workspace + "') != ''",
			Uses: o.CI.Actions.SetupGo,
			With: pinned,
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "go", Name: "Go", BuildMode: "autobuild", Files: workspace, Timeout: o.CI.Timeout},
		},
		Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/", "/**/*"}}},
	}
}
