// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package overlay merges the local files of a repository into its managed files.
//
// A repository adds its own content to the managed file at a path through the local file
// .ergon/local/<path>, which [Read] lists. [Apply] merges it into the rendering of the managed
// file:
//
//   - [Merge] merges a YAML file, by its extension .yml or .yaml. It merges maps key by key, takes
//     the local value for a key that both have, and appends the items of lists.
//   - Any other file is appended to the rendering, after a newline that the rendering lacks.
//
// A managed file has no license header, so neither way copies the license header of a local file
// into it. Apply drops the license header on the first lines of a local file that it appends.
// [Merge] drops the comments that open a local YAML document before an empty line, which contain
// such a header.
//
// The template of a managed file with a comment syntax writes [Header] on its first line. Apply
// replaces it with [MergedHeader], which contains the path of the local file, so a reader of the
// managed file finds where the repository's settings are.
//
// # Dependency position
//
// Imports the standard library and go.yaml.in/yaml/v3. The package
// [go.dokimi.dev/ergon/service/baseline] imports it.
package overlay
