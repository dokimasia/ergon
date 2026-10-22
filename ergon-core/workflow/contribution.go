// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import "fmt"

// OwnPackages is the environment variable to which a step of [Contribution.Review] appends the
// package URLs of the repository's own packages. A package URL is the name that the dependency
// graph of GitHub gives the package. For a directory replace of a Go module, the URL is pkg:golang/
// followed by the path of the replace. The step separates the URLs by commas. It writes a comma
// before its own URLs only when the variable has a value. The dependency review of security.yml
// leaves these packages out of its license check.
const OwnPackages = "ERGON_OWN_PACKAGES"

// Contribution is a producer's part of the workflows of a repository: the setup of a toolchain
// that two languages share, the jobs of ci.yml and of nightly.yml, the setup of the toolchain in
// release.yml, the assets of its releases, the CodeQL analyses of security.yml, the steps before
// its dependency review and the updates of dependabot.yml. The zero value contributes nothing.
type Contribution struct {
	// Setup is the setup of the toolchain of the producer, which the jobs of the toolchain's
	// languages run, or nil. Only a toolchain that two languages share contributes one: a language
	// whose toolchain is its own states the setup in its job.
	Setup *Setup

	// Jobs are the jobs of ci.yml, in the order in which the workflow lists them.
	Jobs []Job

	// Nightly are the jobs of nightly.yml, which runs them on a schedule, in the order in which the
	// workflow lists them. A job runs on the Linux runner of the section github, unless its setup
	// lists runners.
	Nightly []Job

	// Release are the steps that set up the toolchain of the producer in the jobs version and pack
	// of release.yml, which refresh the lockfiles of a release and build its artifacts. They run on
	// the Linux runner of the section github, before the installation of ergon, in the order of the
	// producers.
	Release []Step

	// Assets states that the producer builds release assets in the job pack of release.yml, or is
	// nil.
	Assets *Assets

	// CodeQL are the CodeQL analyses of security.yml.
	CodeQL []CodeQL

	// Review are the steps that the job dependency-review of security.yml runs before the review
	// of the dependencies. The job runs the steps of each producer in the order of the producers. A
	// step that lists the repository's own packages appends their package URLs to [OwnPackages].
	Review []Step

	// Updates are the entries of dependabot.yml.
	Updates []Update
}

// Validate returns the first error of the parts of c, in the order of its fields: the error of
// [Setup.Validate], [Job.Validate] of a job of ci.yml or of nightly.yml, an error that wraps
// [ErrInvalidJob] for a job of ci.yml with App, [Step.Validate] with the position of the release
// step, [Assets.Validate], [CodeQL.Validate], [Step.Validate] with the position of the review step,
// or [Update.Validate], which wraps the sentinel of the part. It returns nil for a contribution
// whose every part is valid, and for the zero value.
func (c *Contribution) Validate() error {
	if c.Setup != nil {
		if err := c.Setup.Validate(); err != nil {
			return err
		}
	}
	for _, jobs := range [][]Job{c.Jobs, c.Nightly} {
		for i := range jobs {
			if err := jobs[i].Validate(); err != nil {
				return err
			}
		}
	}
	for i := range c.Jobs {
		if c.Jobs[i].App {
			return fmt.Errorf("%w: %s of ci.yml creates the token of the GitHub App, which only a job of "+
				"nightly.yml does", ErrInvalidJob, c.Jobs[i].ID)
		}
	}
	for i := range c.Release {
		if err := c.Release[i].Validate(); err != nil {
			return fmt.Errorf("workflow: release step %d: %w", i+1, err)
		}
	}
	if c.Assets != nil {
		if err := c.Assets.Validate(); err != nil {
			return err
		}
	}
	for i := range c.CodeQL {
		if err := c.CodeQL[i].Validate(); err != nil {
			return err
		}
	}
	for i := range c.Review {
		if err := c.Review[i].Validate(); err != nil {
			return fmt.Errorf("workflow: review step %d: %w", i+1, err)
		}
	}
	for i := range c.Updates {
		if err := c.Updates[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
