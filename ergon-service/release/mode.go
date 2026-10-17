// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

// The modes of the release workflow, as the job select-mode of changesets/action v2 outputs them.
const (
	// ModeVersion runs the job version, which opens or updates the version pull request.
	ModeVersion = "version"

	// ModePublish runs the jobs pack and publish.
	ModePublish = "publish"

	// ModeNone runs no other job.
	ModeNone = "none"
)

// SelectMode returns the mode of the release workflow for the release plan p, the publish plan
// publish, and the lockfiles that [Stale] reports for publish. It returns [ModeVersion] for a
// release plan that releases a package and for stale lockfiles, [ModePublish] for a publish plan
// with an entry, and [ModeNone] otherwise. SelectMode ignores a release plan for which
// [Plan.Empty] reports true. A changeset at none or without packages then does not block a publish.
func SelectMode(p *Plan, publish *PublishPlan, stale []string) string {
	switch {
	case !p.Empty(), len(stale) > 0:
		return ModeVersion
	case !publish.Empty():
		return ModePublish
	}
	return ModeNone
}
