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
//   - [MergeTOML] merges a TOML file, by its extension .toml. The local file is a valid TOML
//     document of its own. MergeTOML merges tables key by key, takes the local value for a key that
//     both have, and appends the items of arrays and the tables of arrays of tables.
//   - [MergeJSON] merges a JSON file, by its extension .json. The local file is a valid JSON
//     document of its own. MergeJSON merges objects member by member and takes the local value for
//     a name that both have. It appends the items of arrays, and it writes the merged document as
//     JSON.stringify(value, null, 2) of JavaScript writes it.
//   - Any other file is appended to the rendering, after a newline that the rendering lacks.
//
// A managed file has no license header, so no way copies the license header of a local file into
// it. Apply drops the license header on the first lines of a local file that it appends or merges
// into a TOML file. [Merge] drops the comments that open a local YAML document before an empty line,
// which contain such a header. JSON has no comments, so a local JSON file has no license header.
//
// The template of a managed file with a comment syntax writes [Header] on its first line. Apply
// replaces it with [MergedHeader], which contains the path of the local file, so a reader of the
// managed file finds where the repository's settings are.
//
// # Dependency position
//
// Imports the standard library, with encoding/json/jsontext for the tokens of a JSON document,
// go.yaml.in/yaml/v3, github.com/creachadair/tomledit, which keeps the comments of a TOML document
// through a merge, and github.com/pelletier/go-toml/v2, which checks the rules of TOML. The package
// [go.dokimi.dev/ergon/service/baseline] imports it.
package overlay
