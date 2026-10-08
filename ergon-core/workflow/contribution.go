// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import "fmt"

// Contribution is a producer's part of the workflows of a repository: the setup of a toolchain
// that two languages share, the jobs of ci.yml and of nightly.yml, the setup of the toolchain in
// release.yml, the CodeQL analyses of security.yml and the updates of dependabot.yml. The zero value
// contributes nothing.
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

	// CodeQL are the CodeQL analyses of security.yml.
	CodeQL []CodeQL

	// Updates are the entries of dependabot.yml.
	Updates []Update
}

// Validate returns the first error of the parts of c, in the order of its fields: the error of
// [Setup.Validate], [Job.Validate] of a job of ci.yml or of nightly.yml, [Step.Validate] with the
// position of the release step, [CodeQL.Validate] or [Update.Validate], which wraps the sentinel of
// the part. It returns nil for a contribution whose every part is valid, and for the zero value.
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
	for i := range c.Release {
		if err := c.Release[i].Validate(); err != nil {
			return fmt.Errorf("workflow: release step %d: %w", i+1, err)
		}
	}
	for i := range c.CodeQL {
		if err := c.CodeQL[i].Validate(); err != nil {
			return err
		}
	}
	for i := range c.Updates {
		if err := c.Updates[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
