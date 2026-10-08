// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package options resolves the sections of .ergon.yaml into the options of each producer, and
// writes them back.
//
// A producer of ergon init that is [language.Configurable] declares its options as a struct: one
// field per key, with a yaml tag that names the key and a doc tag that states its meaning, as
// [go.dokimi.dev/ergon/core/option] specifies. A field whose type is a struct is a group of
// options, and every other field is one option: a scalar, a list or a map. [Fields] returns every
// key of the options with its field.
//
// # Resolution
//
// [Resolve] parses .ergon.yaml with viper, and decodes each producer's section into a new value of
// its options, strictly: a key that the struct does not have, and a value of another kind than its
// field, are errors. The value starts at the baseline, so each option takes one value:
//
//   - An option that the section lacks keeps its baseline value.
//   - An option whose value equals the value that the lock records for it takes the baseline value
//     of the installed ergon, so an option that the repository never changed follows the baseline.
//   - Any other option takes the value of the section.
//   - A field with an answer tag takes the value of that answer of ergon init. In a repository with
//     a lock, the section may state the answer or the answer of the lock that a command changes,
//     and any other value is an error.
//
// Resolve then calls the Validate method of every option whose type has one, and the Validate of
// the producer's options last, for the rules between its options. viper reads every key in
// lowercase.
//
// # Writing
//
// [Write] replaces each section of .ergon.yaml with its resolution, under the comment of each doc
// tag, removes the sections of the producers that the repository no longer has, and keeps every
// other key of the file. viper cannot write the file, because its writer drops every comment.
//
// # Errors
//
// An error of the file, of a section or of a value wraps [ErrInvalid] and names the key. Options
// that a producer declares wrong return an error that wraps [ErrDefect].
//
// # Dependency position
//
// Imports the standard library, github.com/spf13/viper, github.com/go-viper/mapstructure/v2,
// go.yaml.in/yaml/v3, [go.dokimi.dev/ergon/core/language] and [go.dokimi.dev/ergon/core/option].
// The packages [go.dokimi.dev/ergon/service/baseline] and [go.dokimi.dev/ergon/service/pin] import
// it.
package options
