// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package spec is the producer of the files of spec in ergon init: the targets of the Makefile, the
// job of ci.yml and the workflow that run spec, the program of stealth-scale/spec that vendors
// conformance definitions.
//
// spec reads its own configuration from .spec.yaml. Its section definition states the repository of
// a definition, and its section implements states the copies of definitions in an implementation.
// A rendering cannot read .spec.yaml, so the key roles of the section spec of .ergon.yaml states
// the role of the repository. A repository without a role does not get a file of spec.
//
// # Options
//
// [Options] is the section spec of .ergon.yaml: the roles, the release of spec, the pin of the
// reusable workflow propose.yml of spec, and the limit of the job spec. [Spec] is a release binary.
// ergon tool run installs its archive and checks it against the digest of the platform.
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
// implementation also gets spec.yml, which calls propose.yml of spec on each weekday with the
// version of the release binary. spec.yml passes the secret ERGON_APP_PRIVATE_KEY by name, because
// GitHub passes secrets through inherit only to a workflow of the same organization or enterprise.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. internal/cli of the
// root module imports it.
package spec
