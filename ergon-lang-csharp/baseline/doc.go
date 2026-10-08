// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of C# of ergon init: the configuration of the analyzers of .NET,
// the fragments of C# of the shared files, the section csharp of .ergon.yaml, and the part of C# of
// the workflows.
//
// [Producer] renders .globalconfig as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-csharp, lint-csharp, test-csharp and audit-csharp with the
// dotnet command, and check-csharp, which requires the targets of the steps that the key check of
// the section names. lint-csharp builds with every rule of the latest analysis level as an error,
// with nullable references and the documentation of every member. audit-csharp restores the
// packages with the audit of NuGet, whose warnings NU1900 to NU1904 fail the build. With a command
// in the key generate, the fragment also renders generate-csharp, which runs it, and
// verify-generate-csharp, which fails when it changes a file. A line ##@ C# starts the group of C#
// in make help.
//
// # Options
//
// [Options] is the section csharp: the steps of the gate, the options of test-csharp, of the
// generators and of audit-csharp, and the key ci of the job check-csharp. [Options.Contribution]
// returns the job check-csharp, the CodeQL analysis of csharp and the updates of the NuGet packages.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
