---
adr: 0052
title: ergon decodes the sections of .ergon.yaml with go.stealthscale.io/cli
status: Accepted
date: 2026-10-16
supersedes: RFC-0001 and RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0052: ergon decodes the sections of .ergon.yaml with go.stealthscale.io/cli

## Status

Accepted

## Context

`options.Resolve` parsed `.ergon.yaml` with viper v1.21.0 and decoded each section into the options of its producer with mapstructure. The decode read each key from the yaml tag of its field, refused a key without a field, and refused a value of another kind, such as a number for a string option.

viper lowercases every key at every depth and has no setting that keeps the case of a key. A release build of ergon showed two options that lose their meaning this way:

- `license.styles` with the key `Dockerfile`. `ergon init new` wrote the key back as `dockerfile`, and `ergon license check` reported `missing Dockerfile`.
- `SHELLCHECK_OPTS` in the `env` of a step of `bash.ci.steps`. `.ergon.yaml` and `ci.yml` received `shellcheck_opts`. The step reads `$SHELLCHECK_OPTS`, and Linux and macOS treat the two names as two variables.

yaml.v3 reads an unquoted `2030-01-01` as a time, and the decode refused a time for a string option. `change-date: 2030-01-01` under `license.parameters` failed `ergon license check` with `expected type 'string', got unconvertible type 'time.Time'`.

The command line of ergon runs on go.stealthscale.io/cli. Its release v0.6.0 adds `cli.Decode`, which decodes a member of a tree into a value of the program. A field without a member keeps its value, so a section decodes into the baseline of its options. `cli.Decode` converts each value from its text, keeps the case of each key and refuses a key without a field.

## Decision

We will decode each section of `.ergon.yaml` with `cli.Decode` of go.stealthscale.io/cli v0.6.0 in place of viper because `cli.Decode` keeps the case of every key and reads each scalar as the file writes it.

- `options.Resolve` parses `.ergon.yaml` with yaml.v3. It compares each option of a section with the record of the lock by the value that yaml.v3 decodes, and removes each option that equals its record.
- The tree that `cli.Decode` reads contains the text of each scalar, so a string option receives `1.20`, `2030-01-01` and `0755` as written. A null member has no value, so its option keeps the baseline.
- `cli.Decode` reads the key of a field from its tag `name` or from the kebab case of its Go name. A group takes its tag `prefix` in place of `name`. The writer and the record still read the key from the yaml tag. Where the two keys differ, six values have the tag `name` and four groups have the tag `prefix`, such as `name:"golangci-lint"` on `GolangCILint`.
- A test writes the section of each producer of ergon with every option set and decodes it with `cli.Decode`. A field whose yaml key differs from the key that `cli.Decode` reads fails the test.
- `ergon-service` requires go-cli in place of viper, mapstructure and seven modules that only viper required. The writer of `.ergon.yaml` still uses yaml.v3, which keeps the comments of the file.

## Alternatives Considered

### Keep viper

viper parses the file and decodes each section strictly through mapstructure. The code of ergon does not change. It lost because viper lowercases every key and has no setting to keep its case. `license.styles` and the `env` of a step then cannot contain a key with an upper-case letter.

### The typed tree of yaml.v3 for cli.Decode

yaml.v3 decodes the document into maps, lists and typed scalars. `cli.Decode` converts each scalar from its text, so ergon does not need a conversion of its own. It lost because yaml.v3 reads an unquoted `1.20` as a float, `2030-01-01` as a time and `0755` as the integer 493. A string option then receives `1.2`, `2030-01-01T00:00:00Z` and `493` without an error.

### The yaml tag in cli.Decode

go-cli would read the key of a field from its yaml tag. The writer, the record and the decode would then read one tag. It lost because go-cli derives the flag, the variable of the environment and the config key of an option from its Go name and its tags `name` and `prefix`. A second set of keys would serve this decode alone, and ten fields of ergon need a second tag.

## Consequences

**Positive:**

- A key of a map keeps its case. `Dockerfile` in `license.styles` matches the file `Dockerfile`, and `SHELLCHECK_OPTS` in the `env` of a step sets that variable.
- A string option takes an unquoted date, version or number as the file writes it.
- The error of an unknown key contains the whole key and the closest key of its struct.

**Negative:**

- A key in another case is an unknown key. viper read `Lint:` under `go` as `lint`.
- `exclude:` without a value keeps the baseline list. viper set an empty list.
- `cli.Decode` converts each value from its text. It accepts `"5"` for an integer, `1` and `"false"` for a bool, and `a,b` for a list of `a` and `b`. viper refused all four. It refuses `5.0` and `0x10` for an integer, and viper accepted both.
- An error contains the first unknown key alone. viper listed every unknown key of a struct.
- A field whose yaml key differs from the kebab case of its Go name has its key in two tags. Only the test keeps the two equal.
- go-cli decodes every exported field, so each exported field of the options of a producer is an option. `options.Fields` returns `ErrDefect` for an exported field tagged `yaml:"-"`, which the writer leaves out.

**Neutral:**

- The root module and `ergon-lang-go` require a release of `ergon-service`, so their `go.mod` lists viper while that release requires it.

## References

| What | Where |
|---|---|
| The case of the keys of viper | <https://github.com/spf13/viper/blob/v1.21.0/README.md#does-viper-support-case-sensitive-keys> |
| The request for `cli.Decode` | <https://github.com/stealth-scale/go-cli/issues/13> |
