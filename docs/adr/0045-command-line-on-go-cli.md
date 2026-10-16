---
adr: 0045
title: The command line of ergon runs on go.stealthscale.io/cli
status: Accepted
date: 2026-10-14
supersedes: RFC-0001, in part
superseded-by: none
rfc: RFC-0001
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0045: The command line of ergon runs on go.stealthscale.io/cli

## Status

Accepted

## Context

The root module ran the command tree of ergon on cobra v1.10.2, and checked with viper v1.21.0 that `.ergon.yaml` or the file of `--config` parsed. Only the root command and the command `completion` ran that check. The commands init, license, release and tool resolved the working directory alone, and read the options of `.ergon.yaml` through the baseline of ergon init. No command read an option from the file of `--config`.

The 24 commands with a handler declared their flags and their arguments by hand, and 7 commands grouped subcommands.

go.stealthscale.io/cli, go-cli for short, runs a command-line program whose options are the fields of Go structs. It derives the flag, the variable of the environment and the help of each option from its field. It adds the flag `--version` and the command `completion`, and its package imports only the standard library. eidos-cli runs on it. Its package `clitest` runs a program of the signature of `App.Main` in a test, and writes the transcript of a list of runs for a golden file.

## Decision

We will run the command line of ergon on go-cli v0.5.1, because go-cli derives the flags, the variables of the environment, the help and the completion of each command from the struct of its options.

- `cli.Program` contains what ergon needs of its process: the working directory, the clock, the cache directory, the random digits, the transport, the platform, the registration of the languages and the version. `Program.Main` has the signature of `App.Main` of go-cli, so `cmd/ergon` passes `gocli.OS()` and a test passes the `Env` of `clitest`.
- The options of each command are the fields of a struct. The flag `--owner` of `ergon init new` reads the variable `ERGON_INIT_NEW_OWNER` too, and a flag replaces the value of the variable. go-cli adds the global flag `--flag-debug`, which writes the value and the source of each option.
- `--config` is removed, because no command reads an option from the file that it named.
- `--version` writes the version, and `-v` is removed, because the flag of the version of go-cli has no short flag.
- A command line that ends at a command without its subcommand, `ergon` alone included, writes the help of that command to the standard error and exits with 2.
- A usage error is one line after the name of the program.
- The help text of each command opens with a summary of one line, followed by its description and an example. The help of its parent lists the command with the summary.
- The tests run ergon through `clitest`, and compare the help and the usage errors of each command with a golden transcript.

## Alternatives Considered

### Keep cobra and viper

Each command would keep its hand-written flags, and the root command would keep the parse check of viper. Roy chose go-cli in their place.

### Keep Run with a Process

`cli.Run(ctx, *Process, register, Version)` would build the `Env` of go-cli from the streams, the arguments and the environment of its `Process`. It lost because `clitest` runs a program of the signature of `App.Main`, and `Process` repeated the fields of `gocli.Env`.

### Keep --config as a config file of go-cli

A field of the type `ConfigFile` would make go-cli read `.ergon.yaml`. It lost because go-cli fails a run for each key of the file that is not the key of an option, and every section of `.ergon.yaml` is such a key.

## Consequences

**Positive:**

- The flag, the variable and the help of each option come from one field.
- The root module requires go-cli in place of cobra and its dependency mousetrap.
- The help lists the variable of the environment of each option.
- A test runs ergon through `clitest` and updates the golden transcript of its help with `-update`.

**Negative:**

- `-v`, `--config` and `ergon help <command>` fail with the exit status 2.
- `ergon` without a command exits with 2.
- A variable such as `ERGON_INIT_SYNC_OWNER` in the environment of a run changes an answer of ergon init sync.
- go-cli requires go.dokimi.dev/assert v0.2.0, so the workspace builds every module of ergon with assert v0.2.0.

## References

| What | Where |
|---|---|
| go-cli | <https://github.com/stealth-scale/go-cli> |
| The summary of a command in the help of its parent | <https://github.com/stealth-scale/go-cli/issues/11> |
| The modules of ergon and their dependencies | RFC-0001 |
