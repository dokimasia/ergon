// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command update-baseline moves the pins of the baselines of ergon's producers to their newest
// releases.
//
//	go run ./internal/cmd/update-baseline [-min-age age] [-major] [-propose]
//
// It runs in the root of ergon's repository, whose working tree must have no changes, because the
// changeset and the pull request cover every change of the tree. It needs GITHUB_TOKEN, and
// -propose needs GITHUB_REPOSITORY as well.
//
// # Steps
//
//  1. It finds the pins of the options of the common files, the GitHub files and the license files,
//     and of the toolchain and the producer of each language of ergon.
//  2. It resolves each pin with [go.dokimi.dev/ergon/service/pin]: the newest stable release of the
//     same major version that its registry published at least -min-age before the run, seven days
//     by default. A module of ergon itself qualifies at once, and -major takes a release of a later
//     major version. It writes a line of each pin that has a newer release, and the error of each
//     pin that does not resolve.
//  3. It lists the packages of ergon's modules with go list, and reads the packages of the release
//     and .changeset/config.json. It then rewrites the Options method of each producer with a newer
//     release through [go.dokimi.dev/ergon/internal/rewrite].
//  4. It runs the tests of each package of ergon's modules with golden files with -update, builds
//     ergon from the sources, and runs its ergon init sync, so the goldens and the managed files of
//     the repository take the new baselines.
//  5. It writes a changeset that releases each package whose producer changed at patch, and names
//     each other changed package at none. Its summary lists the updates.
//  6. With -propose, it commits the changes on the branch ergon-update/<base> through the API of
//     GitHub, and opens or updates the pull request into the base branch of .changeset/config.json.
//     The body lists the updates, the releases of a later major version that the run left out, and
//     the pins that did not resolve.
//
// # Exit status
//
//   - 0 when every pin resolves and every step succeeds
//   - 1 when a step fails, and when a pin does not resolve, after every other step
//   - 2 for a command line that the command refuses
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/changeset],
// [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/version],
// [go.dokimi.dev/ergon/core/workspace], [go.dokimi.dev/ergon/internal/app],
// [go.dokimi.dev/ergon/internal/cli], [go.dokimi.dev/ergon/internal/rewrite],
// [go.dokimi.dev/ergon/service/baseline], [go.dokimi.dev/ergon/service/forge],
// [go.dokimi.dev/ergon/service/pin], [go.dokimi.dev/ergon/service/release] and
// [go.dokimi.dev/ergon/service/vcs].
package main
