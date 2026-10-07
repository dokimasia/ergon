// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package overlay merges the local files of a repository into its managed files.
//
// A repository adds its own content to the managed file at a path through the local file
// .ergon/local/<path>, which [Read] lists. [Apply] merges it into the rendering of the managed
// file:
//
//   - A YAML file, by its extension .yml or .yaml, is merged by [Merge]: maps key by key, with the
//     local value for a key that both have, and lists appended.
//   - Any other file is appended to the rendering, after a newline that the rendering lacks.
//
// The template of a managed file with a comment syntax writes [Header] on its first line. Apply
// replaces it with [MergedHeader], which names the local file, so a reader of the managed file
// finds where the repository's settings are.
//
// # Dependency position
//
// Imports the standard library and go.yaml.in/yaml/v3. The package
// [go.dokimi.dev/ergon/service/baseline] imports it.
package overlay
