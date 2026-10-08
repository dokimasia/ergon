// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Bash of ergon init: the configuration of shellcheck, the
// fragments of Bash of the shared files, the section bash of .ergon.yaml, and the part of Bash of
// the workflows.
//
// [Producer] renders .shellcheckrc as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs lint-bash, which checks every script that git tracks or would
// track and that the pathspecs of the section match, and check-bash. shellcheck is a release binary
// of the section, which ergon tool run installs. Bash has no locked dependencies, so the gate has no
// vulnerability scan. With a command in the key generate, the fragment also renders generate-bash,
// which runs it, and verify-generate-bash, which fails when it changes a file. A line ##@ Bash
// starts the group of Bash in make help.
//
// # Options
//
// [Options] is the section bash: the tools, the paths, the steps of the gate, the options of the
// generators, and the key ci of the job check-bash. [Shellcheck] states where the release of
// shellcheck publishes the asset of each platform. [Options.Contribution] returns the job
// check-bash.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
