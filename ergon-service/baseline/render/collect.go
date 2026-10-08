// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render

import (
	"errors"
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
)

// ErrInvalidContribution is the error for a contribution that a producer declares wrong: a part
// that is not valid, as [workflow.Contribution.Validate] states, a job that names a toolchain whose
// producer contributes no setup, and a job, a CodeQL analysis or an update of a directory that two
// contributions declare. It is a defect of a producer.
var ErrInvalidContribution = errors.New("render: invalid contribution")

// Collect returns the contributions of units to the workflows, in the order of units: every job of
// ci.yml and of nightly.yml, every step of a release, every CodeQL analysis and every update. A job
// that names a toolchain runs the setup that the unit of that name contributes, which Collect sets
// as the job's Setup. A unit that is not a [language.Contributor] contributes nothing, and the
// result has no Setup of its own.
//
// It returns an error that wraps [ErrInvalidContribution] for a contribution that a producer
// declares wrong, and for a job that two contributions declare in the same workflow.
func Collect(units []Unit) (workflow.Contribution, error) {
	setups := map[string]*workflow.Setup{}
	var parts []workflow.Contribution
	var names []string
	for _, u := range units {
		contributor, ok := u.Producer.(language.Contributor)
		if !ok {
			continue
		}
		part := contributor.Contribution(u.Options)
		if err := part.Validate(); err != nil {
			return workflow.Contribution{}, fmt.Errorf("%w: %s: %w", ErrInvalidContribution, u.Name, err)
		}
		if part.Setup != nil {
			setups[u.Name] = part.Setup
		}
		parts = append(parts, part)
		names = append(names, u.Name)
	}
	var all workflow.Contribution
	for i := range parts {
		part := &parts[i]
		var err error
		if all.Jobs, err = jobsOf(all.Jobs, part.Jobs, setups, names[i]); err != nil {
			return workflow.Contribution{}, err
		}
		if all.Nightly, err = jobsOf(all.Nightly, part.Nightly, setups, names[i]); err != nil {
			return workflow.Contribution{}, err
		}
		all.Release = append(all.Release, part.Release...)
		for _, c := range part.CodeQL {
			if slices.ContainsFunc(all.CodeQL, func(d workflow.CodeQL) bool { return d.Language == c.Language }) {
				return workflow.Contribution{}, fmt.Errorf("%w: %s declares the CodeQL analysis of %s again",
					ErrInvalidContribution, names[i], c.Language)
			}
			all.CodeQL = append(all.CodeQL, c)
		}
		for _, u := range part.Updates {
			for _, dir := range u.Directories {
				twice := func(v workflow.Update) bool {
					return v.Ecosystem == u.Ecosystem && slices.Contains(v.Directories, dir)
				}
				if slices.ContainsFunc(all.Updates, twice) {
					return workflow.Contribution{}, fmt.Errorf("%w: %s declares the updates of %s in %s again",
						ErrInvalidContribution, names[i], u.Ecosystem, dir)
				}
			}
			all.Updates = append(all.Updates, u)
		}
	}
	return all, nil
}

// jobsOf returns all with the jobs of the contribution of name appended, each job that names a
// toolchain with the setup of that toolchain from setups. It returns an error that wraps
// [ErrInvalidContribution] for a toolchain without a setup, and for a job whose ID all has.
func jobsOf(all, jobs []workflow.Job, setups map[string]*workflow.Setup, name string) ([]workflow.Job, error) {
	for _, j := range jobs {
		if j.Toolchain != "" {
			setup, ok := setups[j.Toolchain]
			if !ok {
				return nil, fmt.Errorf("%w: the job %s of %s runs the toolchain %s, which contributes no setup",
					ErrInvalidContribution, j.ID, name, j.Toolchain)
			}
			j.Setup = setup
		}
		if slices.ContainsFunc(all, func(k workflow.Job) bool { return k.ID == j.ID }) {
			return nil, fmt.Errorf("%w: %s declares the job %s again", ErrInvalidContribution, name, j.ID)
		}
		all = append(all, j)
	}
	return all, nil
}
