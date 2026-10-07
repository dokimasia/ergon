// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options_test

import (
	"errors"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// written is the section demo at the baseline as Write writes it into an empty file.
const written = `demo:
  # The styles, by the name of a file.
  styles: {}
  # The owner, an answer of ergon init.
  owner: Dokimasia B.V.
  # The tools of the targets.
  tools:
    # The linter of lint-demo.
    lint: golang.org/x/vuln/cmd/govulncheck@v1.8.0
    # The uv of the PyPI packages.
    uv:
      sha256:
        linux/amd64: ` + digest + `
      version: 0.12.23
  # The paths of the targets.
  paths: [./...]
  # The steps of check-demo.
  check: [lint, test]
  # The options of fuzz-demo.
  fuzz:
    match: .
    time: 30s
    args: [-fuzzminimizetime=5s]
  # The pins of the actions, the runners, the versions and the timeout.
  ci:
    actions:
      # The setup of the toolchain.
      setup:
        uses: actions/setup-go
        commit: b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
        release: v7.0.0
    runners: []
    versions: []
    timeout: 30
`

// long is options with a doc that exceeds a line.
type long struct {
	// Text has a doc of 30 words.
	Text string `yaml:"text" doc:"one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twenty-one twenty-two twenty-three twenty-four twenty-five twenty-six twenty-seven twenty-eight twenty-nine thirty"`
}

// Validate returns nil.
func (*long) Validate() error {
	return nil
}

// sparse is options with a field that its encoding omits when it is empty.
type sparse struct {
	// Note is omitted when it is empty.
	Note string `yaml:"note,omitempty" doc:"A note."`
}

// Validate returns nil.
func (*sparse) Validate() error {
	return nil
}

// errMarshal is the error of a value whose encoding fails.
var errMarshal = errors.New("marshal: the value does not encode")

// unencodable is options with a field that YAML cannot encode.
type unencodable struct {
	// Hook is a function.
	Hook func() `yaml:"hook" doc:"A function."`
}

// Validate returns nil.
func (*unencodable) Validate() error {
	return nil
}

// refusal is a value whose encoding in YAML returns errMarshal.
type refusal struct{}

// MarshalYAML returns errMarshal.
func (refusal) MarshalYAML() (any, error) {
	return nil, errMarshal
}

// refusing is options with a field whose encoding fails.
type refusing struct {
	// Value refuses its encoding.
	Value refusal `yaml:"value" doc:"A value that does not encode."`
}

// Validate returns nil.
func (*refusing) Validate() error {
	return nil
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the sections into an empty file under the comment of each doc tag", func(t *testing.T) {
			t.Parallel()
			got, changed, err := options.Write(nil, resolved(t), nil)
			assert.NoError(t, err, "Write")
			assert.Equal(t, string(got), written, "the file")
			assert.True(t, changed, "the change of an empty file")
		})

		t.Run("replaces the value of a section and keeps the other keys with their comments", func(t *testing.T) {
			t.Parallel()
			existing := "# ours\nname: demo\n# the section of demo\ndemo:\n  old: 1\nchecks:\n  coverage: 100\n"
			got, changed, err := options.Write([]byte(existing), resolved(t), nil)
			assert.NoError(t, err, "Write")
			want := "# ours\nname: demo\n# the section of demo\n" + written + "checks:\n  coverage: 100\n"
			assert.Equal(t, string(got), want, "the file")
			assert.True(t, changed, "the change")
		})

		t.Run("removes a dropped section", func(t *testing.T) {
			t.Parallel()
			got, _, err := options.Write([]byte("gone:\n  check: [lint]\nname: demo\n"), resolved(t), []string{"gone"})
			assert.NoError(t, err, "Write")
			assert.Equal(t, string(got), "name: demo\n"+written, "the file")
		})

		t.Run("reports no change for a file that differs in its indentation alone", func(t *testing.T) {
			t.Parallel()
			existing := strings.ReplaceAll(written, "  ", "    ")
			_, changed, err := options.Write([]byte(existing), resolved(t), nil)
			assert.NoError(t, err, "Write")
			assert.False(t, changed, "the change")
		})

		t.Run("reports a change for a value that differs", func(t *testing.T) {
			t.Parallel()
			existing := strings.Replace(written, "timeout: 30", "timeout: 45", 1)
			_, changed, err := options.Write([]byte(existing), resolved(t), nil)
			assert.NoError(t, err, "Write")
			assert.True(t, changed, "the change")
		})

		t.Run("wraps a doc into lines of at most 100 columns", func(t *testing.T) {
			t.Parallel()
			got, _, err := options.Write(nil, []options.Section{{Name: "long", Options: &long{Text: "x"}}}, nil)
			assert.NoError(t, err, "Write")
			assert.Equal(t, string(got), "long:\n"+
				"  # one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen\n"+
				"  # seventeen eighteen nineteen twenty twenty-one twenty-two twenty-three twenty-four twenty-five\n"+
				"  # twenty-six twenty-seven twenty-eight twenty-nine thirty\n"+
				"  text: x\n", "the file")
		})

		t.Run("skips the comment of a field that the encoding omits", func(t *testing.T) {
			t.Parallel()
			got, _, err := options.Write(nil, []options.Section{{Name: "sparse", Options: &sparse{}}}, nil)
			assert.NoError(t, err, "Write")
			assert.Equal(t, string(got), "sparse: {}\n", "the file")
		})

		invalid := []struct {
			name     string
			existing string
		}{
			{name: "returns ErrInvalid for a file that does not parse", existing: "demo: [\n"},
			{name: "returns ErrInvalid for a file whose root is not a mapping", existing: "- demo\n"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, _, err := options.Write([]byte(tt.existing), resolved(t), nil)
				assert.ErrorIs(t, err, options.ErrInvalid, "Write")
			})
		}

		defects := []struct {
			name    string
			options language.Options
		}{
			{name: "returns ErrDefect for options that are not a pointer to a struct", options: valid{}},
			{name: "returns ErrDefect for nil options", options: nil},
			{name: "returns ErrDefect for options with a field without a yaml key", options: &noKey{}},
			{name: "returns ErrDefect for a field of a kind that no document of YAML states", options: &unencodable{}},
			{name: "returns ErrDefect for options whose encoding fails", options: &refusing{}},
		}
		for _, tt := range defects {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, _, err := options.Write(nil, []options.Section{{Name: "odd", Options: tt.options}}, nil)
				assert.ErrorIs(t, err, options.ErrDefect, "Write")
			})
		}
	})
}

// resolved returns the sections that Resolve returns for a repository without .ergon.yaml.
func resolved(t *testing.T) []options.Section {
	t.Helper()
	res, err := options.Resolve(nil, nil, nil, answers(), producers(), []string{name})
	assert.NoError(t, err, "Resolve")
	return res.Sections
}
