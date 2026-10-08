// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// layout is options with an unexported field, a field tagged yaml:"-", and a group that embeds a
// struct inline.
type layout struct {
	// hidden is unexported, so it is no key.
	hidden string

	// Skipped is tagged yaml:"-", so it is no key.
	Skipped string `yaml:"-"`

	// Tool embeds option.Binary inline.
	Tool option.UV `yaml:"tool" doc:"A release binary."`
}

// Validate returns nil.
func (*layout) Validate() error {
	return nil
}

// noKey is options with a field without a yaml key.
type noKey struct {
	// Owner has no yaml tag.
	Owner string
}

// Validate returns nil.
func (*noKey) Validate() error {
	return nil
}

// badGroup is options with a group that has a field without a yaml key.
type badGroup struct {
	// Group has a field without a yaml tag.
	Group struct{ Bad string } `yaml:"group"`
}

// Validate returns nil.
func (*badGroup) Validate() error {
	return nil
}

// inlineText is options with an inline field that is not a struct.
type inlineText struct {
	// Name is inline, and a string.
	Name string `yaml:",inline"`
}

// Validate returns nil.
func (*inlineText) Validate() error {
	return nil
}

// embedded is options that embed a struct without the option inline, from which they take their
// Validate.
type embedded struct {
	valid
}

// inlineBad is options that embed inline a struct with a field without a yaml key.
type inlineBad struct {
	// noKey is inline, and has a field without a yaml key.
	noKey `yaml:",inline"`
}

// word is options whose pointer is no pointer to a struct.
type word string

// Validate returns nil.
func (*word) Validate() error {
	return nil
}

func TestField(t *testing.T) {
	t.Parallel()

	t.Run("Fields", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a group before the keys of the struct that it embeds inline", func(t *testing.T) {
			t.Parallel()
			got, err := options.Fields(&layout{})
			assert.NoError(t, err, "Fields")
			assert.Equal(t, got, []options.Field{
				{Type: reflect.TypeFor[option.UV](), Key: "tool", Doc: "A release binary.", Index: []int{2}},
				{Type: reflect.TypeFor[map[option.Platform]string](), Key: "tool.sha256", Index: []int{2, 0, 0}},
				{Type: reflect.TypeFor[string](), Key: "tool.version", Index: []int{2, 0, 1}},
			}, "the fields")
		})

		t.Run("returns the answer of a field with an answer tag", func(t *testing.T) {
			t.Parallel()
			got, err := options.Fields(&mood{})
			assert.NoError(t, err, "Fields")
			assert.Equal(t, got, []options.Field{
				{Type: reflect.TypeFor[string](), Key: "mood", Answer: "mood", Index: []int{0}},
			}, "the fields")
		})

		defects := []struct {
			name string
			give language.Options
		}{
			{name: "returns ErrDefect for options that are not a pointer", give: valid{}},
			{name: "returns ErrDefect for nil options", give: nil},
			{name: "returns ErrDefect for a pointer to options that are no struct", give: new(word)},
			{name: "returns ErrDefect for a field without a yaml key", give: &noKey{}},
		}
		for _, tt := range defects {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := options.Fields(tt.give)
				assert.ErrorIs(t, err, options.ErrDefect, "Fields")
			})
		}
	})

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("decodes the keys of a struct that a group embeds inline", func(t *testing.T) {
			t.Parallel()
			file := "layout:\n  tool:\n    version: 1.0.0\n    sha256:\n      darwin/arm64: " + digest + "\n"
			res, err := options.Resolve([]byte(file), nil, nil, answers(), layoutProducer(), []string{"layout"})
			assert.NoError(t, err, "Resolve")
			assert.Length(t, res.Sections, 1, "the sections")
			assert.Equal(t, res.Sections[0].Options, language.Options(&layout{hidden: "kept", Tool: option.UV{
				Binary: option.Binary{SHA256: map[option.Platform]string{option.DarwinARM64: digest}, Version: "1.0.0"},
			}}), "the options")
		})

		t.Run("records the keys of a struct inline in a group", func(t *testing.T) {
			t.Parallel()
			res, err := options.Resolve(nil, nil, nil, answers(), layoutProducer(), []string{"layout"})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, res.Record, map[string]any{
				"layout.tool.sha256":  map[option.Platform]string{option.LinuxAMD64: digest},
				"layout.tool.version": "0.12.23",
			}, "the record")
		})

		t.Run("returns ErrInvalid for the key of a field tagged with a dash", func(t *testing.T) {
			t.Parallel()
			_, err := options.Resolve([]byte("layout:\n  skipped: x\n"), nil, nil, answers(), layoutProducer(),
				[]string{"layout"})
			assert.ErrorIs(t, err, options.ErrInvalid, "Resolve")
		})

		defects := []struct {
			name    string
			options func() language.Options
		}{
			{
				name:    "returns ErrDefect for a field without a yaml key",
				options: func() language.Options { return &noKey{} },
			},
			{
				name:    "returns ErrDefect for a group with a field without a yaml key",
				options: func() language.Options { return &badGroup{} },
			},
			{
				name:    "returns ErrDefect for an inline field that is not a struct",
				options: func() language.Options { return &inlineText{} },
			},
			{
				name:    "returns ErrDefect for an embedded field without the option inline",
				options: func() language.Options { return &embedded{} },
			},
			{
				name:    "returns ErrDefect for an inline struct with a field without a yaml key",
				options: func() language.Options { return &inlineBad{} },
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

// layoutProducer returns the producer of the section layout, whose baseline keeps "kept" in the
// unexported field and pins uv for Linux.
func layoutProducer() []options.Producer {
	baseline := func() language.Options {
		return &layout{hidden: "kept", Tool: option.UV{
			Binary: option.Binary{SHA256: map[option.Platform]string{option.LinuxAMD64: digest}, Version: "0.12.23"},
		}}
	}
	return []options.Producer{{Name: "layout", Configurable: odd{options: baseline}}}
}
