// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package option declares the option types that every section of .ergon.yaml composes.
//
// A producer of ergon init declares its options as a struct in its own package. Each field has a
// yaml tag, which names its key, and a [DocTag], which states its meaning. A field's type is a type
// of this package wherever one states the option, so a key means the same in every section and
// each value is checked by one Validate method. A producer adds the options that only it has, and
// checks the rules between its options in the Validate method of its struct.
//
// # Tools
//
// The key tools of a section names one tool per field, by the tool's own name. The type of the
// field states the kind of the tool, and how ergon tool run installs and runs it:
//
//   - [Module]: a Go module, through go install
//   - [PyPI]: a PyPI package, through the [UV] of the same section
//   - [NPM]: an npm package, through npx
//   - [Crate]: a crate, through cargo install
//   - [Maven]: a Maven artifact, through java -jar
//   - [Composer]: a Composer package, through Composer
//   - a [Release]: a release binary, which ergon tool run downloads and checks against the
//     digest that [Binary] pins for its platform, from the releases of its repository on GitHub
//
// A struct tag adds what a tool's package does not state: [ProgramTag] names a program that
// differs from the package, [RunTag] runs a PyPI package in the environment of the project, and
// [ClassifierTag] names the classifier of the jar of a Maven artifact.
//
// A section names the release of software that is no tool of its own as a [Version], such as the
// release of GNU make that a Windows runner installs. Its [SourceTag] states the registry whose
// releases a baseline update reads for it.
//
// # Steps
//
// A [Step] is a target of the Makefile, and [Check] lists the steps of a section's gate. The
// options of a step are [Run], [Generate], [Fuzz], [Bench], [Mutate], [Audit] or [Threshold].
// [Paths] states what the targets work on. The step generate has two targets: generate-<section>
// runs the command of [Generate], and verify-generate-<section>, which the gate runs, fails when
// the command changes a file.
//
// # CI
//
// [CI], [RunnerCI] and [MatrixCI] are the key ci of a section: the pins of the actions of its
// jobs, the runners and the runtime versions of their matrix, and their timeout. The type
// parameter is the producer's struct of the pins of its actions.
//
// # Answers
//
// A field with an [AnswerTag] states an answer of ergon init, such as the owner of a license.
// ergon init writes the answer into the field, so the field follows the lock.
//
// # Errors
//
// Each Validate method returns an error that wraps [ErrInvalid] and names the value. [UV.Asset]
// and every other [Release] return an error that wraps [ErrNoAsset] for a platform without an
// asset.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports the standard library.
package option
