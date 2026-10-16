// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command ergon works on repositories with code in one or more of its languages.
//
//	ergon [--flag-debug] [-h | --help] [--version] <command>
//	ergon init new --owner holder --license identifier --repository owner/name
//		--security-contact address [--name name] [--year year] [--language language]... [--force]
//	ergon init add [--force] language...
//	ergon init remove language...
//	ergon init check [--json]
//	ergon init sync [--name name] [--owner holder] [--license identifier] [--year year]
//		[--repository owner/name] [--security-contact address] [--force]
//	ergon license check [--json]
//	ergon license fix
//	ergon tool run <section>.<tool> [-- argument...]
//	ergon completion bash | fish | powershell | zsh
//
// -h and --help write the help of the command, with the languages, to standard output. --version
// writes the name and the version to standard output. ergon without a command writes the help to
// standard error.
//
// ergon init sets up the repository in the working directory with the baseline of ergon: the
// common files, the GitHub files and the files of each language. It records its answers and the
// digest of each managed file in .ergon/init.lock. ergon init check reports each managed file that
// is missing, edited by hand or outdated, and ergon init sync brings the files to the baseline.
//
// ergon completion writes the completion script of a shell to standard output.
//
// # Environment
//
// Each option of a command also reads a variable of the environment, which joins ERGON, the path
// of the command and the flag, such as ERGON_INIT_NEW_OWNER for --owner of ergon init new. A flag
// replaces the value of the variable. The help of a command lists the variable of each option.
//
// # Signals
//
// SIGINT and SIGTERM cancel the context of the running command.
//
// # Exit status
//
//   - 0 when the command succeeds
//   - 1 when the command fails, such as for a .ergon.yaml that does not parse, or for a managed
//     file that ergon init check finds edited or outdated, and when a language module fails to
//     register, which is a defect of the build
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag, one
//     without a command, or one without a required flag
package main
