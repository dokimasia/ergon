// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command ergon works on repositories with code in one or more of its languages.
//
//	ergon [--config file] [-h | --help] [-v | --version]
//	ergon completion bash | fish | powershell | zsh
//
// -h and --help write the help, with the languages, to standard output, as ergon without an
// argument does. -v and --version write the name and the version to standard output. ergon
// completion writes the completion script of a shell to standard output.
//
// # Configuration
//
// ergon reads its configuration from .ergon.yaml in the working directory, or from the file that
// --config names. The configuration is YAML, whatever the extension of its file. A missing
// .ergon.yaml is no error, and a missing file that --config names is one.
//
// # Signals
//
// SIGINT and SIGTERM cancel the context of the running command.
//
// # Exit status
//
//   - 0 when the command succeeds
//   - 1 when the command fails, such as for a configuration file that does not exist or does not
//     parse, and when a language module fails to register, which is a defect of the build
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag
package main
