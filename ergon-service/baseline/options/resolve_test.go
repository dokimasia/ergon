// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// name is the section of the producer of the cases.
const name = "demo"

// digest is a valid digest of the cases.
const digest = "9167d72b3319674b6303c4cbe071854bba13ebdf3d76b1a7cbdc175471fb66d6"

// setup is the pin of the setup action of the cases.
var setup = workflow.Action{
	Uses:    "actions/setup-go",
	Commit:  "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
	Release: "v7.0.0",
}

// tools are the tools of the options of the cases.
type tools struct {
	// Lint is a Go module.
	Lint option.Module `yaml:"lint" doc:"The linter of lint-demo."`

	// UV is a release binary.
	UV option.UV `yaml:"uv" doc:"The uv of the PyPI packages."`
}

// actions are the pins of the actions of the cases.
type actions struct {
	// Setup is the setup of the toolchain.
	Setup workflow.Action `yaml:"setup" doc:"The setup of the toolchain."`
}

// demo is the options of the producer of the cases: an answer, tools, paths, steps, the options of
// a step, a map and the key ci.
type demo struct {
	Styles map[string]string        `yaml:"styles" doc:"The styles, by the name of a file."`
	Owner  string                   `yaml:"owner"  doc:"The owner, an answer of ergon init."                                 answer:"owner"`
	Tools  tools                    `yaml:"tools"  doc:"The tools of the targets."`
	Paths  option.Paths             `yaml:"paths"  doc:"The paths of the targets."`
	Check  option.Check             `yaml:"check"  doc:"The steps of check-demo."`
	Fuzz   option.Fuzz              `yaml:"fuzz"   doc:"The options of fuzz-demo."`
	CI     option.MatrixCI[actions] `yaml:"ci"     doc:"The pins of the actions, the runners, the versions and the timeout."`
}

// Validate returns an error for a step of check that demo does not have.
func (d *demo) Validate() error {
	return d.Check.Only(option.StepLint, option.StepTest, option.StepFuzz)
}

// configurable is the producer of the cases, whose options are demo at the baseline.
type configurable struct{}

// Options returns a new demo at the baseline.
func (configurable) Options() language.Options {
	return baseline()
}

// odd is a producer whose options have a field of the case's choice, and a step of check.
type odd struct {
	// options returns the options of the producer.
	options func() language.Options
}

// Options returns the options of o.
func (o odd) Options() language.Options {
	return o.options()
}

// valid is options that are no struct pointer, and that implement Validate and accept every value.
type valid struct{}

// Validate returns nil.
func (valid) Validate() error {
	return nil
}

// mood is options with an answer that ergon init does not have.
type mood struct {
	// Mood states the answer mood.
	Mood string `yaml:"mood" answer:"mood"`
}

// Validate returns nil.
func (*mood) Validate() error {
	return nil
}

// yearText is options with an answer of another type: the languages as a string.
type yearText struct {
	// Year states the answer languages.
	Year string `yaml:"year" answer:"languages"`
}

// Validate returns nil.
func (*yearText) Validate() error {
	return nil
}

// unset is options with a list and a map that are nil at the baseline.
type unset struct {
	// List is a list of the baseline.
	List []string `yaml:"list" doc:"A list, nil at the baseline."`

	// Map is a map of the baseline.
	Map map[string]string `yaml:"map" doc:"A map, nil at the baseline."`
}

// Validate returns nil.
func (*unset) Validate() error {
	return nil
}

// named is options with the answer name, the first answer of ergon init.
type named struct {
	// Name states the answer name.
	Name string `yaml:"name" answer:"name"`
}

// Validate returns nil.
func (*named) Validate() error {
	return nil
}

// moodGroup is options with an answer in a group that ergon init does not have.
type moodGroup struct {
	// Group contains the answer mood.
	Group struct {
		Mood string `yaml:"mood" answer:"mood"`
	} `yaml:"group"`
}

// Validate returns nil.
func (*moodGroup) Validate() error {
	return nil
}

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the baseline with the answers for a file without the section", func(t *testing.T) {
			t.Parallel()
			res, err := options.Resolve(nil, nil, nil, answers(), producers(), []string{name})
			assert.NoError(t, err, "Resolve")
			want := baseline()
			want.Owner = "Dokimasia B.V."
			assert.Equal(t, res.Sections, []options.Section{{Name: name, Options: want}}, "the sections")
			assert.Empty(t, res.Drop, "the dropped sections")
		})

		t.Run("records the baseline value of each option but the answers", func(t *testing.T) {
			t.Parallel()
			res, err := options.Resolve(nil, nil, nil, answers(), producers(), []string{name})
			assert.NoError(t, err, "Resolve")
			b := baseline()
			assert.Equal(t, res.Record, map[string]any{
				"demo.styles":                   b.Styles,
				"demo.tools.lint":               b.Tools.Lint,
				"demo.tools.uv.sha256":          b.Tools.UV.SHA256,
				"demo.tools.uv.version":         b.Tools.UV.Version,
				"demo.paths":                    b.Paths,
				"demo.check":                    b.Check,
				"demo.fuzz.match":               b.Fuzz.Match,
				"demo.fuzz.time":                b.Fuzz.Time,
				"demo.fuzz.args":                b.Fuzz.Args,
				"demo.ci.actions.setup.uses":    setup.Uses,
				"demo.ci.actions.setup.commit":  setup.Commit,
				"demo.ci.actions.setup.release": setup.Release,
				"demo.ci.runners":               b.CI.Runners,
				"demo.ci.versions":              b.CI.Versions,
				"demo.ci.timeout":               b.CI.Timeout,
			}, "the record")
		})

		relicensed := answers()
		relicensed.Owner = "Someone Else"
		tests := []struct {
			name     string
			file     string
			recorded map[string]any
			previous *language.Answers
			want     func(*demo)
		}{
			{
				name: "takes the value of the section for an option that the lock does not record",
				file: "demo:\n  fuzz:\n    time: 60s\n",
				want: func(d *demo) { d.Fuzz.Time = "60s" },
			},
			{
				name:     "takes the baseline for an option whose value equals the record",
				file:     "demo:\n  fuzz:\n    time: 10s\n",
				recorded: map[string]any{"demo.fuzz.time": "10s"},
				want:     func(*demo) {},
			},
			{
				name:     "takes the value of the section for an option that differs from the record",
				file:     "demo:\n  fuzz:\n    time: 20s\n",
				recorded: map[string]any{"demo.fuzz.time": "10s"},
				want:     func(d *demo) { d.Fuzz.Time = "20s" },
			},
			{
				name:     "takes the baseline for a list equal to the record",
				file:     "demo:\n  check: [lint]\n",
				recorded: map[string]any{"demo.check": []any{"lint"}},
				want:     func(*demo) {},
			},
			{
				name: "replaces the list of the baseline with a shorter list",
				file: "demo:\n  check: [test]\n",
				want: func(d *demo) { d.Check = option.Check{option.StepTest} },
			},
			{
				name: "replaces the map of the baseline with the map of the section",
				file: "demo:\n  tools:\n    uv:\n      sha256:\n        linux/arm64: " + digest + "\n",
				want: func(d *demo) { d.Tools.UV.SHA256 = map[option.Platform]string{option.LinuxARM64: digest} },
			},
			{
				name:     "takes the baseline for a map equal to the record",
				file:     "demo:\n  tools:\n    uv:\n      sha256:\n        linux/arm64: " + digest + "\n",
				recorded: map[string]any{"demo.tools.uv.sha256": map[string]any{"linux/arm64": digest}},
				want:     func(*demo) {},
			},
			{
				name: "keeps the baseline of the options that a group of the section lacks",
				file: "demo:\n  ci:\n    timeout: 45\n",
				want: func(d *demo) { d.CI.Timeout = 45 },
			},
			{
				name: "takes the answer for a field that the section of a repository without a lock states",
				file: "demo:\n  owner: Someone Else\n",
				want: func(*demo) {},
			},
			{
				name:     "takes the answer for a field that states the answer of the lock",
				file:     "demo:\n  owner: Someone Else\n",
				previous: relicensed,
				want:     func(*demo) {},
			},
			{
				name:     "takes the answer for a field that states the answer",
				file:     "demo:\n  owner: Dokimasia B.V.\n",
				previous: relicensed,
				want:     func(*demo) {},
			},
			{
				name: "reads a key of a map in lowercase",
				file: "demo:\n  styles:\n    Dockerfile: Hashtag\n",
				want: func(d *demo) { d.Styles = map[string]string{"dockerfile": "Hashtag"} },
			},
			{
				name: "returns the baseline for a section without a value",
				file: "demo:\n",
				want: func(*demo) {},
			},
			{
				name: "keeps a key of the file that is no section",
				file: "checks:\n  coverage: 100\n",
				want: func(*demo) {},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res, err := options.Resolve([]byte(tt.file), tt.recorded, tt.previous, answers(), producers(),
					[]string{name})
				assert.NoError(t, err, "Resolve")
				want := baseline()
				want.Owner = "Dokimasia B.V."
				tt.want(want)
				assert.Length(t, res.Sections, 1, "the sections")
				assert.Equal(t, res.Sections[0].Options, language.Options(want), "the options of demo")
			})
		}

		t.Run("takes the answer name for a field that states it", func(t *testing.T) {
			t.Parallel()
			p := options.Producer{Name: name, Configurable: odd{options: func() language.Options { return &named{} }}}
			res, err := options.Resolve(nil, nil, nil, answers(), []options.Producer{p}, []string{name})
			assert.NoError(t, err, "Resolve")
			assert.Length(t, res.Sections, 1, "the sections")
			assert.Equal(t, res.Sections[0].Options, language.Options(&named{Name: "demo"}), "the options")
		})

		t.Run("records an empty list and an empty map for a nil list and a nil map", func(t *testing.T) {
			t.Parallel()
			p := options.Producer{Name: name, Configurable: odd{options: func() language.Options { return &unset{} }}}
			res, err := options.Resolve(nil, nil, nil, answers(), []options.Producer{p}, []string{name})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, res.Record, map[string]any{
				"demo.list": []string{},
				"demo.map":  map[string]string{},
			}, "the record")
		})

		t.Run("takes the baseline for an empty list and map that equal the record of nil ones", func(t *testing.T) {
			t.Parallel()
			p := options.Producer{Name: name, Configurable: odd{options: func() language.Options { return &unset{} }}}
			recorded := map[string]any{"demo.list": []any{}, "demo.map": map[string]any{}}
			res, err := options.Resolve([]byte("demo:\n  list: []\n  map: {}\n"), recorded, nil, answers(),
				[]options.Producer{p}, []string{name})
			assert.NoError(t, err, "Resolve")
			assert.Length(t, res.Sections, 1, "the sections")
			assert.Equal(t, res.Sections[0].Options, language.Options(&unset{}), "the options")
		})

		t.Run("drops the section of a producer that the record states", func(t *testing.T) {
			t.Parallel()
			recorded := map[string]any{"gone.check": []any{"lint"}}
			res, err := options.Resolve([]byte("gone:\n  check: [lint]\n"), recorded, nil, answers(), producers(),
				[]string{name, "gone"})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, res.Drop, []string{"gone"}, "the dropped sections")
		})

		t.Run("leaves out an option that the producer no longer has at the value of its record", func(t *testing.T) {
			t.Parallel()
			recorded := map[string]any{"demo.tools.gone": "example.com/gone@v1.0.0", "demo.fuzz.time": "10s"}
			file := "demo:\n  tools:\n    gone: example.com/gone@v1.0.0\n  fuzz:\n    time: 10s\n"
			res, err := options.Resolve([]byte(file), recorded, nil, answers(), producers(), []string{name})
			assert.NoError(t, err, "Resolve")
			want := baseline()
			want.Owner = "Dokimasia B.V."
			assert.Equal(t, res.Sections, []options.Section{{Name: name, Options: want}}, "the sections")
		})

		t.Run("returns ErrInvalid for an option that the producer no longer has at another value", func(t *testing.T) {
			t.Parallel()
			recorded := map[string]any{"demo.tools.gone": "example.com/gone@v1.0.0"}
			file := "demo:\n  tools:\n    gone: example.com/gone@v2.0.0\n"
			_, err := options.Resolve([]byte(file), recorded, nil, answers(), producers(), []string{name})
			assert.ErrorIs(t, err, options.ErrInvalid, "Resolve")
			assert.Contains(t, err.Error(), "gone", "the error")
		})

		t.Run("does not modify the record", func(t *testing.T) {
			t.Parallel()
			recorded := map[string]any{"demo.fuzz.time": "10s"}
			_, err := options.Resolve([]byte("demo:\n  fuzz:\n    time: 10s\n"), recorded, nil, answers(), producers(),
				[]string{name})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, recorded, map[string]any{"demo.fuzz.time": "10s"}, "the record after Resolve")
		})

		invalid := []struct {
			name string
			file string
			want string
		}{
			{
				name: "returns ErrInvalid for a file that does not parse",
				file: "demo: [\n",
				want: "options: invalid .ergon.yaml: ",
			},
			{
				name: "returns ErrInvalid for a section that is not a mapping",
				file: "demo: 5\n",
				want: "demo, which must be a mapping",
			},
			{
				name: "returns ErrInvalid for a key that the section does not have",
				file: "demo:\n  fuzz:\n    tme: 1s\n",
				want: "tme",
			},
			{
				name: "returns ErrInvalid for a value of another kind",
				file: "demo:\n  fuzz:\n    time: [1s]\n",
				want: "demo: ",
			},
			{
				name: "returns ErrInvalid for a value that its option does not accept",
				file: "demo:\n  fuzz:\n    time: 0s\n",
				want: "demo.fuzz: option: invalid value: time",
			},
			{
				name: "returns ErrInvalid for an error of the Validate of the options",
				file: "demo:\n  check: [race]\n",
				want: "demo: option: invalid value: check names race",
			},
			{
				name: "returns ErrInvalid for the section of no producer that the record does not state",
				file: "gone:\n  check: [lint]\n",
				want: "gone, which is the section of no producer of the repository",
			},
			{
				name: "returns ErrInvalid for a field that states neither the answer nor the answer of the lock",
				file: "demo:\n  owner: Someone Else\n",
				want: `demo.owner "Someone Else", which ergon init sync --owner sets, differs from the answer ` +
					`"Dokimasia B.V." of ergon init`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := options.Resolve([]byte(tt.file), nil, answers(), answers(), producers(),
					[]string{name, "gone"})
				assert.ErrorIs(t, err, options.ErrInvalid, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}

		t.Run("returns the error of the option for a value that its option does not accept", func(t *testing.T) {
			t.Parallel()
			_, err := options.Resolve(
				[]byte("demo:\n  fuzz:\n    time: 0s\n"),
				nil,
				nil,
				answers(),
				producers(),
				[]string{name},
			)
			assert.ErrorIs(t, err, option.ErrInvalid, "Resolve")
		})

		defects := []struct {
			name    string
			options func() language.Options
		}{
			{
				name:    "returns ErrDefect for options that are not a pointer to a struct",
				options: func() language.Options { return valid{} },
			},
			{
				name:    "returns ErrDefect for an answer that ergon init does not have",
				options: func() language.Options { return &mood{} },
			},
			{
				name:    "returns ErrDefect for an answer of another type",
				options: func() language.Options { return &yearText{} },
			},
			{
				name:    "returns ErrDefect for an answer in a group that ergon init does not have",
				options: func() language.Options { return &moodGroup{} },
			},
		}
		for _, tt := range defects {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p := options.Producer{Name: name, Configurable: odd{options: tt.options}}
				_, err := options.Resolve(nil, nil, nil, answers(), []options.Producer{p}, []string{name})
				assert.ErrorIs(t, err, options.ErrDefect, "Resolve")
			})
		}
	})
}

// baseline returns a new demo at the baseline, without its answer.
func baseline() *demo {
	return &demo{
		Styles: map[string]string{},
		Tools: tools{
			Lint: "golang.org/x/vuln/cmd/govulncheck@v1.8.0",
			UV: option.UV{Binary: option.Binary{
				SHA256:  map[option.Platform]string{option.LinuxAMD64: digest},
				Version: "0.12.23",
			}},
		},
		Paths: option.Paths{"./..."},
		Check: option.Check{option.StepLint, option.StepTest},
		Fuzz:  option.Fuzz{Match: ".", Time: "30s", Args: []string{"-fuzzminimizetime=5s"}},
		CI: option.MatrixCI[actions]{
			Actions:  actions{Setup: setup},
			Runners:  []string{},
			Versions: []string{},
			Timeout:  30,
		},
	}
}

// producers returns the producers of the cases: demo alone.
func producers() []options.Producer {
	return []options.Producer{{Name: name, Configurable: configurable{}}}
}

// answers returns the answers of the cases.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Owner:           "Dokimasia B.V.",
		License:         spdx.MIT,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
		Languages:       []workspace.Language{"demo"},
		Year:            2026,
	}
}
