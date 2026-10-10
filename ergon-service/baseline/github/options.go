// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// cronFields is the number of fields of a cron expression of GitHub Actions: the minute, the hour,
// the day of the month, the month and the day of the week.
const cronFields = 5

// cronField matches one field of a cron expression, such as 3, */15, 1-5 or MON.
var cronField = regexp.MustCompile(`^[0-9A-Za-z*,/?-]+$`)

// The names of a job of a matrix, after the job's own name: the runner, and the runtime version
// where the matrix has versions.
const (
	runnerSuffix  = " (${{ matrix.os }})"
	versionSuffix = " (${{ matrix.os }}, ${{ matrix.version }})"
)

// ToolsCache starts the key of each cache of GitHub Actions that keeps the tool directory of ergon
// for a job of ci.yml or nightly.yml. The key continues with the system, the architecture, the job
// and the runtime version of a matrix, each followed by a dash, and ends in the digest of the files
// of the key, which contains no dash.
const ToolsCache = "ergon-tools-"

// The step that keeps the tool directory of ergon in the cache of GitHub Actions, in a job that
// runs tools: its name and its ID, the directory, which is ergon/tools in the cache directory of the
// user on each system, as os.UserCacheDir returns it, its prefix without and with the runtime
// version of a matrix, and the files of its key. The key ends in the digest of .ergon.yaml, the lock
// and the version files of the setup of the job, so it changes with each option, each version of
// ergon and each version of the toolchain. On a miss, the prefix restores the newest cache of the
// job, and ergon tool run installs only the tools whose versions that cache lacks.
const (
	toolsStep          = "Keep the tools of ergon"
	toolsID            = "ergon-tools"
	toolsPath          = "${{ runner.os == 'Windows' && '~/AppData/Local/ergon/tools' || runner.os == 'macOS' && '~/Library/Caches/ergon/tools' || '~/.cache/ergon/tools' }}"
	toolsPrefix        = ToolsCache + "${{ runner.os }}-${{ runner.arch }}-${{ github.job }}-"
	toolsVersionPrefix = ToolsCache + "${{ runner.os }}-${{ runner.arch }}-${{ github.job }}-${{ matrix.version }}-"
	toolsFiles         = "'.ergon.yaml', '.ergon/init.lock'"
)

// The last step of a job that runs tools, which removes each tool that the options do not name
// from the tool directory before the cache saves it: its name, its condition, and its command. It
// runs only when the key of the cache missed, because actions/cache does not save after a hit.
const (
	pruneToolsStep = "Prune the tools of ergon"
	pruneToolsIf   = "steps." + toolsID + ".outputs.cache-hit != 'true'"
	pruneToolsRun  = "ergon tool prune"
)

// The job of nightly.yml that deletes the caches of the tools of ergon that newer caches of the same
// job replaced: its key, its name, the name of its step, and the command of the step.
const (
	pruneID   = "prune-tools"
	pruneName = "Prune the tool caches"
	pruneStep = "Delete the tool caches that newer caches replaced"
	pruneRun  = "ergon tool ci prune"
)

// The variable of the environment from which ergon reads the token of GitHub, and the expression
// of the token of the run, which GitHub Actions replaces before the step runs.
const (
	authVariable   = "GITHUB_TOKEN"
	authExpression = "${{ github.token }}"
)

// Job is a job of ci.yml as its template renders it: the values of a [workflow.Job] that the
// section github, the job's setup and the checkout determine. The template renders the checkout and
// the installation of GNU make before Setup, and the installation of ergon between Setup and Steps.
type Job struct {
	// ID is the key of the job in the workflow, such as check-go.
	ID string

	// Name is the name of the job in the checks of a pull request. The name of a job of a matrix
	// ends in the runner and the runtime version, such as "Go (${{ matrix.os }}, ${{ matrix.version
	// }})".
	Name string

	// If is the condition of the job, or empty.
	If string

	// Guard is the condition of every step after the checkout, hashFiles('<files>') != '', for a job
	// whose setup states files, and empty for any other job.
	Guard string

	// Permissions are the access of the job's GITHUB_TOKEN to each scope.
	Permissions map[string]string

	// Env are the environment variables of the job, by name.
	Env map[string]string

	// Runners are the runner images of the job's matrix, and nil for a check of text, which runs on
	// the Linux runner of the section.
	Runners []string

	// Versions are the runtime versions of the job's matrix, or nil.
	Versions []string

	// Setup are the setup steps of the job, each with Guard before its own condition, which run
	// before the installation of ergon.
	Setup []workflow.Step

	// Steps are the steps of the job, each with Guard before its own condition. For a job that runs
	// tools, the step that keeps the tools of ergon in the cache comes first, and the step that
	// prunes them comes last.
	Steps []workflow.Step

	// Timeout is the limit of the job in minutes.
	Timeout int

	// History reports that the checkout fetches every commit.
	History bool

	// Make reports that the job installs GNU make, as every job with a setup does.
	Make bool

	// Ergon reports that the job installs ergon, as a job with a setup or a job that runs ergon does.
	Ergon bool
}

// Options are the options of the section github of .ergon.yaml.
type Options struct {
	// Runners are the runner images of the jobs that run on every runner.
	Runners option.Runners `yaml:"runners" doc:"The runner images of the jobs that run on every runner, each pinned to a version of its system. The key ci.runners of a section lists a part of them."`

	// Linux is the runner image of the jobs that check text.
	Linux string `yaml:"linux" doc:"The runner image of the jobs that check text and of the analyses of CodeQL, one of runners."`

	// Make is the release of GNU make that a Windows runner installs.
	Make option.Version `yaml:"make" source:"chocolatey:make" doc:"The release of GNU make that the action setup-make installs from Chocolatey on a Windows runner, whose image has no make."`

	// Nightly are the options of nightly.yml.
	Nightly Nightly `yaml:"nightly" doc:"The options of nightly.yml, which runs the steps under the key nightly of each language on a schedule, each in a job of its own."`

	// CI are the pins of the actions of the GitHub files, and the limit of their jobs.
	CI option.CI[Actions] `yaml:"ci" doc:"The pins of the actions of the workflows, and the limit in minutes of each job of the GitHub files."`
}

// Nightly are the options of nightly.yml.
type Nightly struct {
	// Schedule is when the workflow runs, as a cron expression of GitHub Actions in UTC.
	Schedule string `yaml:"schedule" doc:"When nightly.yml runs, as a cron expression of five fields in UTC, such as 0 3 * * * for 03:00 each day."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a Schedule that is not five fields of
// digits, letters and the characters * , - / and ?, separated by spaces.
func (n Nightly) Validate() error {
	fields := strings.Fields(n.Schedule)
	malformed := func(f string) bool { return !cronField.MatchString(f) }
	if len(fields) != cronFields || slices.ContainsFunc(fields, malformed) {
		return fmt.Errorf("%w: nightly.schedule %q, which is no cron expression of five fields", option.ErrInvalid,
			n.Schedule)
	}
	return nil
}

// Actions are the pins of the actions of the GitHub files.
type Actions struct {
	// Checkout checks out the repository in every job.
	Checkout workflow.Action `yaml:"checkout" doc:"The action that checks out the repository in every job of every workflow."`

	// CodeQL analyzes the sources, and uploads the results of the Scorecard.
	CodeQL workflow.Action `yaml:"codeql" doc:"The actions of CodeQL. init and analyze run the analysis of each language in codeql.yml, and upload-sarif uploads the results of the Scorecard."`

	// DependencyReview reviews the dependencies that a pull request changes.
	DependencyReview workflow.Action `yaml:"dependency-review" doc:"The action that reviews the dependencies that a pull request changes, in the job dependency-review of security.yml."`

	// Scorecard runs the checks of the OpenSSF Scorecard.
	Scorecard workflow.Action `yaml:"scorecard" doc:"The action of the OpenSSF Scorecard, in the weekly job scorecard of security.yml."`

	// UploadArtifact passes a file of a job of release.yml to a later job.
	UploadArtifact workflow.Action `yaml:"upload-artifact" doc:"The action that uploads the publish plan of the job select-mode and the artifacts of the job pack of release.yml."`

	// DownloadArtifact receives a file of an earlier job of release.yml.
	DownloadArtifact workflow.Action `yaml:"download-artifact" doc:"The action that downloads the publish plan in the job pack, and the artifacts in the job publish of release.yml."`

	// Cache keeps the tools of ergon of a job between its runs.
	Cache workflow.Action `yaml:"cache" doc:"The action that restores the tool directory of ergon before the steps of each job of ci.yml that runs tools, and saves it after a run that succeeds."`

	// CreateGitHubAppToken creates the token of a GitHub App that opens the version pull request.
	CreateGitHubAppToken workflow.Action `yaml:"create-github-app-token" doc:"The action that creates a token of the GitHub App of the variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY in version.yml, whose pull request then runs its checks without an approval, and in the job homebrew of release.yml, which commits the casks of a release to the tap."`

	// Attest attests the assets of a release with SLSA build provenance.
	Attest workflow.Action `yaml:"attest" doc:"The action that attests the assets of a release with SLSA build provenance in the job pack of release.yml, which GitHub stores in its attestation API."`
}

// Validate returns an error that wraps [option.ErrInvalid] for no runner, and for a Linux that
// Runners does not list. The command checks each option by the Validate method of its type before
// it calls Validate.
func (o *Options) Validate() error {
	if len(o.Runners) == 0 {
		return fmt.Errorf("%w: runners, which lists no runner for the jobs that run on every runner", option.ErrInvalid)
	}
	if !slices.Contains(o.Runners, o.Linux) {
		return fmt.Errorf("%w: linux %q, which runners does not list", option.ErrInvalid, o.Linux)
	}
	return nil
}

// Jobs returns the jobs of c as ci.yml renders them for o, in the order of c:
//
//   - A check of text runs on the runner Linux, without a matrix.
//   - A job with a setup runs on the runners of its setup, or on every runner of o when the setup
//     lists none, and on the versions of its setup. It takes the timeout and the environment of its
//     setup, runs the setup steps before the installation of ergon, and installs GNU make and
//     ergon. A setup that states files guards every step after the checkout.
//   - Any other job runs on every runner of o, and installs ergon when it runs ergon.
//   - A job that runs tools keeps the tool directory of ergon in the cache with the action Cache,
//     in a step before its own steps. The key of the cache names the runtime version of a matrix
//     with versions, and covers the version files of its setup. Its prefix restores the newest
//     cache of the job when the key misses. After its own steps, a step runs ergon tool prune when
//     the key missed, so the cache saves only the tools that the options name.
//
// It returns an error that wraps [option.ErrInvalid] for a job that runs on a runner that Runners
// does not list. Jobs reads c and does not modify it: a guarded step is a copy. A Job shares its
// maps and its unguarded steps with c, and its runners with o.
func (o *Options) Jobs(c *workflow.Contribution) ([]Job, error) {
	jobs := make([]Job, 0, len(c.Jobs))
	for i := range c.Jobs {
		j := &c.Jobs[i]
		job := Job{
			ID:          j.ID,
			Name:        j.Name,
			If:          j.If,
			Permissions: j.Permissions,
			Steps:       j.Steps,
			Timeout:     j.Timeout,
			History:     j.History,
			Ergon:       j.Ergon,
		}
		if !j.Text {
			job.Runners = o.Runners
		}
		if s := j.Setup; s != nil {
			if len(s.Runners) > 0 {
				job.Runners = s.Runners
			}
			job.Versions = s.Versions
			job.Env = s.Env
			job.Setup = s.Steps
			job.Timeout = s.Timeout
			job.Make, job.Ergon = true, true
			if s.Files != "" {
				job.Guard = "hashFiles('" + s.Files + "') != ''"
			}
		}
		if j.Tools {
			prefix := toolsPrefix
			if len(job.Versions) > 0 {
				prefix = toolsVersionPrefix
			}
			files := toolsFiles
			if j.Setup != nil && j.Setup.VersionFiles != "" {
				files += ", '" + j.Setup.VersionFiles + "'"
			}
			tools := workflow.Step{
				Name: toolsStep,
				ID:   toolsID,
				Uses: o.CI.Actions.Cache,
				With: map[string]string{
					"path":         toolsPath,
					"key":          prefix + "${{ hashFiles(" + files + ") }}",
					"restore-keys": prefix,
				},
			}
			prune := workflow.Step{Name: pruneToolsStep, If: pruneToolsIf, Run: []string{pruneToolsRun}}
			job.Steps = slices.Concat([]workflow.Step{tools}, j.Steps, []workflow.Step{prune})
		}
		for _, r := range job.Runners {
			if !slices.Contains(o.Runners, r) {
				return nil, fmt.Errorf("%w: the job %s runs on %s, which runners of the section github does not list",
					option.ErrInvalid, j.ID, r)
			}
		}
		if len(job.Versions) > 0 {
			job.Name += versionSuffix
		} else if job.Runners != nil {
			job.Name += runnerSuffix
		}
		// A guard comes with a setup. The guarded steps are copies, so the conditions change no step
		// of c, and no step of a setup that the jobs of two languages share.
		if job.Guard != "" {
			job.Setup, job.Steps = guard(job.Setup, job.Guard), guard(job.Steps, job.Guard)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// NightlyJobs returns the nightly jobs of c as nightly.yml renders them for o, in the order of c. It
// renders each job as [Options.Jobs] does, but a job without a setup that lists runners runs on the
// runner Linux: without a matrix and without a runner in its name, unless its setup lists versions,
// which a matrix of Linux runs. When a job of c keeps the tools of ergon in the cache, the last job
// is prune-tools. It runs ergon tool ci prune on the runner Linux with the permission actions:
// write. The command deletes each cache of the tools that a newer cache of the same job replaced.
// NightlyJobs returns the errors of Options.Jobs.
func (o *Options) NightlyJobs(c *workflow.Contribution) ([]Job, error) {
	nightly := c.Nightly
	tools := func(j workflow.Job) bool { return j.Tools }
	if slices.ContainsFunc(c.Jobs, tools) || slices.ContainsFunc(c.Nightly, tools) {
		nightly = slices.Concat(nightly, []workflow.Job{{
			ID:          pruneID,
			Name:        pruneName,
			Timeout:     o.CI.Timeout,
			Permissions: map[string]string{"actions": "write", "contents": "read"},
			Ergon:       true,
			Steps: []workflow.Step{{
				Name: pruneStep,
				Env:  map[string]string{authVariable: authExpression},
				Run:  []string{pruneRun},
			}},
		}})
	}
	jobs, err := o.Jobs(&workflow.Contribution{Jobs: nightly})
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		s := nightly[i].Setup
		switch {
		case s != nil && len(s.Runners) > 0:
		case s != nil && len(s.Versions) > 0:
			jobs[i].Runners = []string{o.Linux}
		default:
			jobs[i].Runners, jobs[i].Name = nil, nightly[i].Name
		}
	}
	return jobs, nil
}

// guard returns a copy of steps in which condition comes before the condition of each step, so each
// step runs only when condition is true. The copies share their maps with steps.
func guard(steps []workflow.Step, condition string) []workflow.Step {
	guarded := slices.Clone(steps)
	for k := range guarded {
		step := &guarded[k]
		if step.If == "" {
			step.If = condition
		} else {
			step.If = condition + " && (" + step.If + ")"
		}
	}
	return guarded
}
