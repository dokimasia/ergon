// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// The names of a job of a matrix, after the job's own name: the runner, and the runtime version
// where the matrix has versions.
const (
	runnerSuffix  = " (${{ matrix.os }})"
	versionSuffix = " (${{ matrix.os }}, ${{ matrix.version }})"
)

// Job is a job of ci.yml as its template renders it: the values of a [workflow.Job] that the
// section github, the job's setup and the checkout determine. The template renders the checkout,
// the installation of GNU make and the installation of ergon before Steps.
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

	// Steps are the setup steps and the steps of the job, each with Guard before its own condition.
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
	Make option.Version `yaml:"make" doc:"The release of GNU make that the action setup-make installs from Chocolatey on a Windows runner, whose image has no make."`

	// CI are the pins of the actions of the GitHub files, and the limit of their jobs.
	CI option.CI[Actions] `yaml:"ci" doc:"The pins of the actions of the workflows, and the limit in minutes of each job of the GitHub files."`
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
//     setup, runs the setup steps before its own, and installs GNU make and ergon. A setup that
//     states files guards every step after the checkout.
//   - Any other job runs on every runner of o, and installs ergon when it runs ergon.
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
			job.Steps = slices.Concat(s.Steps, j.Steps)
			job.Timeout = s.Timeout
			job.Make, job.Ergon = true, true
			if s.Files != "" {
				job.Guard = "hashFiles('" + s.Files + "') != ''"
			}
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
		// A guard comes with a setup, whose steps are a new slice of copies, so the conditions change
		// no step of c.
		if job.Guard != "" {
			for k := range job.Steps {
				step := &job.Steps[k]
				if step.If == "" {
					step.If = job.Guard
				} else {
					step.If = job.Guard + " && (" + step.If + ")"
				}
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
