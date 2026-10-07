// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render_test

import (
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/render"
)

func TestFunction(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			template string
			data     any
			want     string
		}{
			{
				name:     "words writes plain words as they are",
				template: "{{% words .Data %}}",
				data:     []string{"./...", "-count=1"},
				want:     "./... -count=1",
			},
			{
				name:     "words writes a list of a defined type",
				template: "{{% words .Data %}}",
				data:     option.Paths{"*.sh"},
				want:     "'*.sh'",
			},
			{
				name:     "words quotes a word with a space",
				template: "{{% words .Data %}}",
				data:     []string{"my scripts"},
				want:     "'my scripts'",
			},
			{
				name:     "words escapes a single quote of a word",
				template: "{{% words .Data %}}",
				data:     []string{"it's"},
				want:     `'it'\''s'`,
			},
			{
				name:     "words escapes the dollar sign and the number sign for make",
				template: "{{% words .Data %}}",
				data:     []string{"$HOME#x"},
				want:     `'$$HOME\#x'`,
			},
			{
				name:     "words writes nothing for an empty list",
				template: "{{% words .Data %}}",
				data:     []string{},
				want:     "",
			},
			{name: "words writes an array", template: "{{% words .Data %}}", data: [2]string{"a", "b"}, want: "a b"},
			{
				name:     "make escapes the dollar sign and the number sign",
				template: "{{% make .Data %}}",
				data:     "a$b#c",
				want:     `a$$b\#c`,
			},
			{
				name:     "make writes a string of a defined type",
				template: "{{% make .Data %}}",
				data:     option.StepLint,
				want:     "lint",
			},
			{name: "yaml writes a boolean", template: "{{% yaml .Data %}}", data: false, want: "false"},
			{name: "yaml writes an integer", template: "{{% yaml .Data %}}", data: 30, want: "30"},
			{
				name:     "yaml writes a plain string as it is",
				template: "{{% yaml .Data %}}",
				data:     "temurin",
				want:     "temurin",
			},
			{
				name:     "yaml writes a file name that starts with a full stop as it is",
				template: "{{% yaml .Data %}}",
				data:     ".java-version",
				want:     ".java-version",
			},
			{name: "yaml writes a path as it is", template: "{{% yaml .Data %}}", data: "/", want: "/"},
			{
				name:     "yaml writes words separated by single spaces as they are",
				template: "{{% yaml .Data %}}",
				data:     "Check the managed files",
				want:     "Check the managed files",
			},
			{
				name:     "yaml quotes words separated by two spaces",
				template: "{{% yaml .Data %}}",
				data:     "Check  files",
				want:     `"Check  files"`,
			},
			{
				name:     "yaml quotes a string that ends in a space",
				template: "{{% yaml .Data %}}",
				data:     "Check ",
				want:     `"Check "`,
			},
			{
				name:     "yaml quotes words with a colon",
				template: "{{% yaml .Data %}}",
				data:     "Check: files",
				want:     `"Check: files"`,
			},
			{name: "yaml quotes a version", template: "{{% yaml .Data %}}", data: "1.27.1", want: `"1.27.1"`},
			{name: "yaml quotes a glob", template: "{{% yaml .Data %}}", data: "**/go.sum", want: `"**/go.sum"`},
			{
				name:     "yaml quotes a word that YAML 1.1 reads as a boolean",
				template: "{{% yaml .Data %}}",
				data:     "On",
				want:     `"On"`,
			},
			{
				name:     "yaml quotes a number that starts with a full stop",
				template: "{{% yaml .Data %}}",
				data:     ".5",
				want:     `".5"`,
			},
			{
				name:     "yaml quotes an expression",
				template: "{{% yaml .Data %}}",
				data:     "${{ matrix.version }}",
				want:     `"${{ matrix.version }}"`,
			},
			{
				name:     "yaml quotes a string that spans lines",
				template: "{{% yaml .Data %}}",
				data:     "a\nb",
				want:     `"a\nb"`,
			},
			{name: "yaml quotes the empty string", template: "{{% yaml .Data %}}", data: "", want: `""`},
			{
				name:     "yaml writes a list in flow style",
				template: "{{% yaml .Data %}}",
				data:     []string{"ubuntu-26.04", "1.27"},
				want:     `[ubuntu-26.04, "1.27"]`,
			},
			{name: "yaml writes an empty list", template: "{{% yaml .Data %}}", data: []string{}, want: "[]"},
			{
				name:     "steps writes an action with its name, its condition and its inputs",
				template: "{{% steps .Data %}}",
				data: []workflow.Step{{
					Name: "Set up Java", If: "hashFiles('.java-version') != ''", Uses: setupJava,
					With: map[string]string{"java-version-file": ".java-version", "distribution": "temurin"},
				}},
				want: "\n      - name: Set up Java\n        if: \"hashFiles('.java-version') != ''\"\n        uses: " +
					setupJava.String() + "\n        with:\n          distribution: temurin\n" +
					"          java-version-file: .java-version",
			},
			{
				name:     "steps writes a command with its id and its environment",
				template: "{{% steps .Data %}}",
				data: []workflow.Step{
					{ID: "mode", Env: map[string]string{"TOKEN": "${{ github.token }}"}, Run: []string{"a", "b c"}},
				},
				want: "\n      - id: mode\n        env:\n          TOKEN: \"${{ github.token }}\"\n        run: |\n" +
					"          a\n          b c",
			},
			{
				name:     "steps writes each step as an item",
				template: "{{% steps .Data %}}",
				data:     []workflow.Step{{Uses: setupJava}, {Run: []string{"java -version"}}},
				want:     "\n      - uses: " + setupJava.String() + "\n      - run: |\n          java -version",
			},
			{
				name:     "steps writes nothing for no step",
				template: "{{% steps .Data %}}",
				data:     []workflow.Step{},
				want:     "",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				u := render.Unit{Name: "go", Producer: fixture{data: tt.data, templates: fstest.MapFS{
					"managed/x.tmpl": {Data: []byte(tt.template + "\n")},
				}}}
				files, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
				assert.NoError(t, err, "Render")
				assert.Length(t, files, 1, "the files")
				assert.Equal(t, string(files[0].Content), tt.want+"\n", "the rendering")
			})
		}

		invalid := []struct {
			name     string
			template string
			data     any
		}{
			{name: "words returns ErrInvalidTemplate for a string", template: "{{% words .Data %}}", data: "a b"},
			{
				name:     "words returns ErrInvalidTemplate for a list of integers",
				template: "{{% words .Data %}}",
				data:     []int{1},
			},
			{name: "make returns ErrInvalidTemplate for an integer", template: "{{% make .Data %}}", data: 1},
			{
				name:     "yaml returns ErrInvalidTemplate for a map",
				template: "{{% yaml .Data %}}",
				data:     map[string]string{},
			},
			{
				name:     "yaml returns ErrInvalidTemplate for a list of maps",
				template: "{{% yaml .Data %}}",
				data:     []map[string]string{{}},
			},
			{name: "steps returns ErrInvalidTemplate for a string", template: "{{% steps .Data %}}", data: "run"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				u := render.Unit{Name: "go", Producer: fixture{data: tt.data, templates: fstest.MapFS{
					"managed/x.tmpl": {Data: []byte(tt.template + "\n")},
				}}}
				_, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
				assert.ErrorIs(t, err, render.ErrInvalidTemplate, "Render")
			})
		}
	})
}
