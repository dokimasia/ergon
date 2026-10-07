// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workflow declares a producer's part of the GitHub workflows of a repository.
//
// A producer of ergon init returns its part as a [Contribution]: the jobs of ci.yml, the steps that
// set up its toolchain in release.yml, the CodeQL analyses of security.yml and the updates of
// dependabot.yml. The producer of the GitHub files renders the contributions of every producer into
// those files, so no toolchain appears in its templates.
//
//   - A [Job] is a job of ci.yml: the steps that follow the checkout, the permissions, and the
//     runners. A check job of a language runs the [Setup] of its toolchain before its own steps.
//   - A [Setup] states how a job sets up a toolchain: the files whose presence runs the job, the
//     runners and the runtime versions of its matrix, its timeout, and its setup steps.
//   - A [Step] runs an [Action] at the commit of a release, or a command of bash.
//   - A [CodeQL] analysis and an [Update] of Dependabot each state one language or one package
//     manager.
//
// A toolchain that two languages share contributes the setup of their jobs, and each language
// names the toolchain in [Job.Toolchain]. A language whose toolchain is its own sets [Job.Setup]
// itself.
//
// # Expressions
//
// A condition, such as [Job.If] or [Step.If], is an expression of GitHub Actions without the
// ${{ }} around it. The renderer writes it as a YAML scalar. A pattern of files, such as
// [Setup.Files], is an argument of hashFiles in single quotes, so it contains no single quote.
//
// # Errors
//
// Each type has a Validate method. It returns an error that wraps the sentinel of its type,
// [ErrInvalidAction], [ErrInvalidStep], [ErrInvalidJob], [ErrInvalidCodeQL] or [ErrInvalidUpdate],
// for the first value that a workflow cannot contain.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports the standard library.
package workflow
