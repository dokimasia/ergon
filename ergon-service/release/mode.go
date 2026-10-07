// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "go.dokimi.dev/ergon/core/changeset"

// The modes of the release workflow, as the job select-mode of changesets/action v2 outputs them.
const (
	// ModeVersion runs the job version, which opens or updates the version pull request.
	ModeVersion = "version"

	// ModePublish runs the jobs pack and publish.
	ModePublish = "publish"

	// ModeNone runs no other job.
	ModeNone = "none"
)

// SelectMode returns the mode of the release workflow for the changesets sets of the repository and
// the publish plan plan: [ModeVersion] for a repository with changesets, [ModePublish] for a plan
// with an entry, and [ModeNone] otherwise.
func SelectMode(sets []changeset.Changeset, plan *PublishPlan) string {
	switch {
	case len(sets) > 0:
		return ModeVersion
	case !plan.Empty():
		return ModePublish
	}
	return ModeNone
}
