// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package spec is the producer of the files of spec in ergon init: the targets of the Makefile and
// the jobs of ci.yml and nightly.yml that run spec, the program of stealth-scale/spec that vendors
// conformance definitions.
//
// spec reads its own configuration from .spec.yaml. Its section definition states the repository of
// a definition, and its section implements states the copies of definitions in an implementation.
// A rendering cannot read .spec.yaml, so the key roles of the section spec of .ergon.yaml states
// the role of the repository. A repository without a role does not get a file of spec.
//
// # Options
//
// [Options] is the section spec of .ergon.yaml: the roles, the release of spec, and the limit of the
// jobs of spec. [Spec] is a release binary. ergon tool run installs its archive and checks it
// against the digest of the platform.
//
// # Makefile
//
// The fragment of the Makefile runs spec through ergon tool run spec.spec:
//
//   - An implementation gets check-spec, which runs spec check. The target check depends on it.
//   - A definition gets generate-spec, which writes the manifest with spec manifest, and
//     verify-generate-spec, which runs spec manifest --check. The targets generate and check depend
//     on them.
//
// # Workflows
//
// [Producer.Contribution] returns the job spec of ci.yml, a check of text with a step for each
// role: spec check --remote for an implementation, and spec manifest --check for a definition. An
// implementation also gets the job propose-spec of nightly.yml. The job runs spec propose for the
// repository. It creates the token of the GitHub App of ergon and passes the token to spec in
// GITHUB_TOKEN. spec does not read a variable or a secret of ergon.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. internal/cli of the
// root module imports it.
package spec
