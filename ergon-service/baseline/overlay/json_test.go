// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline/overlay"
)

// biome is the rendering of the JSON cases in the layout of the merge: a string, an object and an
// array of objects.
const biome = "{\n" +
	"  \"$schema\": \"https://biomejs.dev/schemas/2.5.15/schema.json\",\n" +
	"  \"linter\": {\n" +
	"    \"rules\": {\n" +
	"      \"style\": \"error\"\n" +
	"    }\n" +
	"  },\n" +
	"  \"overrides\": [\n" +
	"    {\n" +
	"      \"includes\": [\n" +
	"        \"**\"\n" +
	"      ]\n" +
	"    }\n" +
	"  ]\n" +
	"}\n"

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("MergeJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			base string
			over string
			want string
		}{
			{
				name: "returns the first document for a second document without members",
				base: biome,
				over: "{}\n",
				want: biome,
			},
			{
				name: "adds a member that the object of the first document lacks after its members",
				base: biome,
				over: `{"files": {"includes": ["**", "!!**/.ergon"]}}`,
				want: "{\n" +
					"  \"$schema\": \"https://biomejs.dev/schemas/2.5.15/schema.json\",\n" +
					"  \"linter\": {\n" +
					"    \"rules\": {\n" +
					"      \"style\": \"error\"\n" +
					"    }\n" +
					"  },\n" +
					"  \"overrides\": [\n" +
					"    {\n" +
					"      \"includes\": [\n" +
					"        \"**\"\n" +
					"      ]\n" +
					"    }\n" +
					"  ],\n" +
					"  \"files\": {\n" +
					"    \"includes\": [\n" +
					"      \"**\",\n" +
					"      \"!!**/.ergon\"\n" +
					"    ]\n" +
					"  }\n" +
					"}\n",
			},
			{
				name: "merges the objects member by member",
				base: biome,
				over: `{"linter": {"rules": {"suspicious": "error"}}}`,
				want: "{\n" +
					"  \"$schema\": \"https://biomejs.dev/schemas/2.5.15/schema.json\",\n" +
					"  \"linter\": {\n" +
					"    \"rules\": {\n" +
					"      \"style\": \"error\",\n" +
					"      \"suspicious\": \"error\"\n" +
					"    }\n" +
					"  },\n" +
					"  \"overrides\": [\n" +
					"    {\n" +
					"      \"includes\": [\n" +
					"        \"**\"\n" +
					"      ]\n" +
					"    }\n" +
					"  ]\n" +
					"}\n",
			},
			{
				name: "appends the items of an array",
				base: biome,
				over: `{"overrides": [{"includes": ["src/**"]}]}`,
				want: "{\n" +
					"  \"$schema\": \"https://biomejs.dev/schemas/2.5.15/schema.json\",\n" +
					"  \"linter\": {\n" +
					"    \"rules\": {\n" +
					"      \"style\": \"error\"\n" +
					"    }\n" +
					"  },\n" +
					"  \"overrides\": [\n" +
					"    {\n" +
					"      \"includes\": [\n" +
					"        \"**\"\n" +
					"      ]\n" +
					"    },\n" +
					"    {\n" +
					"      \"includes\": [\n" +
					"        \"src/**\"\n" +
					"      ]\n" +
					"    }\n" +
					"  ]\n" +
					"}\n",
			},
			{
				name: "replaces a value with the value of the second document",
				base: `{"a": "x", "b": 1}`,
				over: `{"a": "y"}`,
				want: "{\n  \"a\": \"y\",\n  \"b\": 1\n}\n",
			},
			{
				name: "replaces an object with a value of another type",
				base: `{"a": {"b": 1}}`,
				over: `{"a": [1]}`,
				want: "{\n  \"a\": [\n    1\n  ]\n}\n",
			},
			{
				name: "replaces an array with a value of another type",
				base: `{"a": [1]}`,
				over: `{"a": null}`,
				want: "{\n  \"a\": null\n}\n",
			},
			{
				name: "replaces the first document with a second document of another type",
				base: `{"a": 1}`,
				over: `[true]`,
				want: "[\n  true\n]\n",
			},
			{
				name: "keeps the text of each number and string",
				base: `{"a": 1.50, "b": "caf\u00e9 & <tea>"}`,
				over: `{"c": [1e3, "\u0041"]}`,
				want: "{\n  \"a\": 1.50,\n  \"b\": \"caf\\u00e9 & <tea>\",\n  \"c\": [\n    1e3,\n    \"\\u0041\"\n  ]\n}\n",
			},
			{
				name: "writes an empty object and an empty array on one line",
				base: `{"a": {}}`,
				over: `{"b": []}`,
				want: "{\n  \"a\": {},\n  \"b\": []\n}\n",
			},
			{name: "returns the first document for an empty second document", base: biome, over: " \n", want: biome},
			{
				name: "returns the second document for an empty first document",
				base: "",
				over: `{"a": 1}`,
				want: `{"a": 1}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := overlay.MergeJSON([]byte(tt.base), []byte(tt.over))
				assert.NoError(t, err, "MergeJSON")
				assert.Equal(t, string(got), tt.want, "the merged document")
			})
		}

		refusals := []struct {
			name string
			base string
			over string
		}{
			{name: "returns an error for a first document that does not parse", base: `{"a": }`, over: `{}`},
			{name: "returns an error for a second document that does not parse", base: `{}`, over: `{"a": }`},
			{name: "returns an error for a document that ends inside its value", base: `{}`, over: `{"a": [1`},
			{name: "returns an error for a name that an object has twice", base: `{}`, over: `{"a": 1, "a": 2}`},
			{name: "returns an error for a comment", base: `{}`, over: "{\"a\": 1} // local\n"},
			{name: "returns an error for a value after the value of the document", base: `{}`, over: `{"a": 1} {}`},
			{name: "returns an error for a value after a value that is not an object", base: `{}`, over: `1 2`},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := overlay.MergeJSON([]byte(tt.base), []byte(tt.over))
				assert.HasError(t, err, "MergeJSON")
				assert.HasPrefix(t, err.Error(), "parse: ", "the error")
			})
		}
	})
}
