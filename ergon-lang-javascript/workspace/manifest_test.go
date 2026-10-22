// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

// manifestB is the package.json of probe-b in the cases. It has a version, a dependency in each
// section, a key that a release does not read, and spaces and line breaks of its own.
const manifestB = `{
  "name": "@probe/b",
  "version" :  "0.2.0",
  "private": false,
  "scripts": {"build": "tsc"},
  "dependencies": {"@probe/a": "workspace:^", "left-pad": "^1.3.0"},
  "optionalDependencies": {"@probe/a": "^0.1.0"},
  "peerDependencies": {"@probe/a": ">=0.1.0"},
  "devDependencies": {"@probe/a": "workspace:*"}
}
`

// bom is the byte order mark of UTF-8.
const bom = "\xef\xbb\xbf"

func TestManifest(t *testing.T) {
	t.Parallel()

	t.Run("ParseManifest", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the name and the version with the span of the version", func(t *testing.T) {
			t.Parallel()
			m, err := jsworkspace.ParseManifest("package.json", []byte(manifestB))
			assert.NoError(t, err, "ParseManifest")
			assert.Equal(t, m.Name, "@probe/b", "the name")
			assert.Equal(t, m.Version, "0.2.0", "the version")
			assert.Equal(t, manifestB[m.VersionSpan.Start:m.VersionSpan.End], `"0.2.0"`, "the text of the span")
		})

		t.Run("returns each requirement with its section and the span of its string", func(t *testing.T) {
			t.Parallel()
			m, err := jsworkspace.ParseManifest("package.json", []byte(manifestB))
			assert.NoError(t, err, "ParseManifest")
			got := make([]string, 0, len(m.Requirements))
			for _, e := range m.Requirements {
				got = append(got, string(e.Kind)+" "+e.Name+" "+e.Text+" "+manifestB[e.Span.Start:e.Span.End])
			}
			assert.Equal(t, got, []string{
				`runtime @probe/a workspace:^ "workspace:^"`,
				`runtime left-pad ^1.3.0 "^1.3.0"`,
				`optional @probe/a ^0.1.0 "^0.1.0"`,
				`peer @probe/a >=0.1.0 ">=0.1.0"`,
				`dev @probe/a workspace:* "workspace:*"`,
			}, "the requirements")
		})

		t.Run("returns the spans of a file that starts with a byte order mark", func(t *testing.T) {
			t.Parallel()
			data := bom + manifestB
			m, err := jsworkspace.ParseManifest("package.json", []byte(data))
			assert.NoError(t, err, "ParseManifest")
			assert.Equal(t, data[m.VersionSpan.Start:m.VersionSpan.End], `"0.2.0"`, "the text of the span")
		})

		tests := []struct {
			name        string
			give        string
			wantPrivate bool
			wantGlobs   []string
		}{
			{name: "reports a private package", give: `{"name": "a", "private": true}`, wantPrivate: true},
			{name: "reports a package that is not private", give: `{"name": "a", "private": false}`},
			{
				name:      "returns the globs of a list of workspaces",
				give:      `{"name": "root", "workspaces": ["packages/*", "tools/cli"]}`,
				wantGlobs: []string{"packages/*", "tools/cli"},
			},
			{
				name:      "returns the globs of the packages of an object of workspaces",
				give:      `{"name": "root", "workspaces": {"packages": ["packages/**"], "nohoist": ["**/x"]}}`,
				wantGlobs: []string{"packages/**"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				m, err := jsworkspace.ParseManifest("package.json", []byte(tt.give))
				assert.NoError(t, err, "ParseManifest")
				assert.Equal(t, m.Private, tt.wantPrivate, "the private flag")
				assert.Equal(t, m.Workspaces, tt.wantGlobs, "the globs of the workspaces")
			})
		}

		t.Run("returns the unescaped text of a string", func(t *testing.T) {
			t.Parallel()
			m, err := jsworkspace.ParseManifest("package.json", []byte(`{"name": "@probe\/a"}`))
			assert.NoError(t, err, "ParseManifest")
			assert.Equal(t, m.Name, "@probe/a", "the name")
		})

		t.Run("returns no requirement for a package without sections", func(t *testing.T) {
			t.Parallel()
			m, err := jsworkspace.ParseManifest("package.json", []byte(`{"name": "a"}`))
			assert.NoError(t, err, "ParseManifest")
			assert.Empty(t, m.Requirements, "the requirements")
			assert.Equal(t, m.Version, "", "the version")
		})

		invalid := []struct {
			name     string
			give     string
			wantText string
		}{
			{name: "returns ErrManifest for an empty file", give: ``, wantText: "EOF"},
			{
				name:     "returns ErrManifest for a file that is not an object",
				give:     `["a"]`,
				wantText: "the top-level value, which is not an object",
			},
			{
				name:     "returns ErrManifest for an object that does not end",
				give:     `{"name": "a"`,
				wantText: "unexpected EOF",
			},
			{
				name:     "returns ErrManifest for data after the object",
				give:     `{"name": "a"} {}`,
				wantText: "package.json, which has data after its object",
			},
			{
				name:     "returns ErrManifest for a member that the object repeats",
				give:     `{"name": "a", "name": "b"}`,
				wantText: "duplicate object member name",
			},
			{
				name:     "returns ErrManifest for a member that does not parse",
				give:     `{"scripts": [}`,
				wantText: "invalid character",
			},
			{
				name:     "returns ErrManifest for a name that is not a string",
				give:     `{"name": 1}`,
				wantText: "name, which is not a string",
			},
			{name: "returns ErrManifest for a name that does not end", give: `{"name": "a`, wantText: "unexpected EOF"},
			{
				name:     "returns ErrManifest for a version that is not a string",
				give:     `{"version": 1}`,
				wantText: "version, which is not a string",
			},
			{
				name:     "returns ErrManifest for a private flag that is not a boolean",
				give:     `{"private": "yes"}`,
				wantText: "private, which is not a boolean",
			},
			{
				name:     "returns ErrManifest for a private flag that does not parse",
				give:     `{"private": tru}`,
				wantText: "invalid character",
			},
			{
				name:     "returns ErrManifest for a section that is not an object",
				give:     `{"dependencies": ["a"]}`,
				wantText: "dependencies, which is not an object",
			},
			{
				name:     "returns ErrManifest for a section that does not start",
				give:     `{"dependencies": `,
				wantText: "unexpected EOF",
			},
			{
				name:     "returns ErrManifest for a requirement that is not a string",
				give:     `{"dependencies": {"a": 1}}`,
				wantText: "dependencies.a, which is not a string",
			},
			{
				name:     "returns ErrManifest for workspaces that are neither a list nor an object",
				give:     `{"workspaces": "packages/*"}`,
				wantText: "workspaces, which is neither a list nor an object",
			},
			{
				name:     "returns ErrManifest for workspaces that do not start",
				give:     `{"workspaces": `,
				wantText: "unexpected EOF",
			},
			{
				name:     "returns ErrManifest for a member of workspaces that does not parse",
				give:     `{"workspaces": {"nohoist": [}}`,
				wantText: "invalid character",
			},
			{
				name:     "returns ErrManifest for packages that are not a list",
				give:     `{"workspaces": {"packages": "a"}}`,
				wantText: "workspaces.packages, which is not a list",
			},
			{
				name:     "returns ErrManifest for packages that do not start",
				give:     `{"workspaces": {"packages": `,
				wantText: "unexpected EOF",
			},
			{
				name:     "returns ErrManifest for a glob that is not a string",
				give:     `{"workspaces": [1]}`,
				wantText: "workspaces, which has an item that is not a string",
			},
			{
				name:     "returns ErrManifest for a list of workspaces that does not end",
				give:     `{"workspaces": ["a"`,
				wantText: "unexpected EOF",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := jsworkspace.ParseManifest("package.json", []byte(tt.give))
				assert.ErrorIs(t, err, jsworkspace.ErrManifest, "ParseManifest")
				assert.Contains(t, err.Error(), tt.wantText, "the text of the error")
			})
		}
	})
}
