// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"go.dokimi.dev/ergon/core/language"
)

// ErrInvalid is the error for .ergon.yaml that does not parse, a section with a key that its
// producer's struct does not have or a value that does not decode into its field, a value that its
// option does not accept, and the section of a producer that the repository does not have. Its
// text names the key.
var ErrInvalid = errors.New("options: invalid .ergon.yaml")

// ErrDefect is the error for options that a producer declares wrong: options that are not a
// pointer to a struct, a field without a yaml key, an inline field that is not a struct, and a
// field with an answer tag that names no answer of ergon init or whose type the answer does not
// convert to. It is a defect of the producer.
var ErrDefect = errors.New("options: invalid options of a producer")

// configType is the format of .ergon.yaml for viper.
const configType = "yaml"

// Producer is a configurable producer and the name of its section of .ergon.yaml.
type Producer struct {
	// Configurable returns the options of the producer at the baseline.
	Configurable language.Configurable

	// Name is the name of the section, such as go.
	Name string
}

// Section is the resolved options of one producer.
type Section struct {
	// Options are the resolved options, which a producer renders.
	Options language.Options

	// Name is the name of the section.
	Name string
}

// Resolution is the options of every producer of a repository.
type Resolution struct {
	// Record is the baseline value of each option, by its key in .ergon.yaml, such as go.fuzz.time,
	// which the lock records. It has no field that states an answer.
	Record map[string]any

	// Sections are the options of each producer, in the order of the producers.
	Sections []Section

	// Drop are the sections of .ergon.yaml whose producers the repository no longer has.
	Drop []string
}

// Resolve returns the options of producers from file, the content of .ergon.yaml, from recorded,
// the record of the lock, and from a, the answers of ergon init. previous are the answers of the
// lock, which a may change, and nil for a repository without a lock. viper parses file and decodes
// each section into a new value of its producer's options, which starts with the baseline value of
// every option that the section lacks. An option whose value in file equals the value that recorded
// records for it takes its baseline value, so an option that the repository never changed follows
// the baseline of the installed ergon. Such an option that the producer no longer has is left out,
// so the baseline of the installed ergon removes it. A field with an answer tag takes the answer's
// value of a.
// names are the names of every section that a producer of the catalog can have: a section of names
// that no producer of producers states is dropped when recorded records an option of it.
//
// A section that states an answer with the value of a or of previous, which ergon init wrote before
// a changed it, is consistent with the lock. ergon init writes the answers of a repository without
// a lock from a alone, so for a nil previous a section may state any value of an answer.
//
// It returns an error that wraps [ErrInvalid], and names the key, for a file that does not parse, a
// section that is not a mapping, a key that a section does not have, a value that does not decode
// into its field, an answer that a section states with a value that is neither the answer of a nor
// of a non-nil previous, a value whose Validate method returns an error, an error of the Validate of
// a producer's options, and a section of names that the repository has no producer of and that
// recorded records no option of. It returns an error that wraps [ErrDefect] for options that a
// producer declares wrong. Resolve reads file, recorded, previous and a, and modifies none of them.
func Resolve(file []byte, recorded map[string]any, previous, a *language.Answers, producers []Producer,
	names []string,
) (Resolution, error) {
	v := viper.New()
	v.SetConfigType(configType)
	if err := v.ReadConfig(bytes.NewReader(file)); err != nil {
		return Resolution{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	res := Resolution{Record: map[string]any{}}
	for _, p := range producers {
		o, err := resolve(v, &p, recorded, previous, a, res.Record)
		if err != nil {
			return Resolution{}, err
		}
		res.Sections = append(res.Sections, Section{Name: p.Name, Options: o})
	}
	keys := slices.Collect(maps.Keys(recorded))
	for _, name := range names {
		own := slices.ContainsFunc(producers, func(p Producer) bool { return p.Name == name })
		if own || !v.InConfig(name) {
			continue
		}
		if !slices.ContainsFunc(keys, func(k string) bool { return strings.HasPrefix(k, name+".") }) {
			return Resolution{}, fmt.Errorf("%w: %s, which is the section of no producer of the repository", ErrInvalid,
				name)
		}
		res.Drop = append(res.Drop, name)
	}
	return res, nil
}

// resolve returns the options of p, as [Resolve] states, and adds the baseline value of each of its
// options to record.
func resolve(v *viper.Viper, p *Producer, recorded map[string]any, previous, a *language.Answers,
	record map[string]any,
) (language.Options, error) {
	o := p.Configurable.Options()
	fs, err := Fields(o)
	if err != nil {
		return nil, fmt.Errorf("%w: the section %s", err, p.Name)
	}
	options := reflect.ValueOf(o).Elem()
	if err := answer(options, fs, a); err != nil {
		return nil, err
	}
	for _, f := range fs {
		if f.Type.Kind() != reflect.Struct && f.Answer == "" {
			record[p.Name+"."+f.Key] = stated(options.FieldByIndex(f.Index))
		}
	}
	if v.InConfig(p.Name) {
		section, ok := v.Get(p.Name).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: %s, which must be a mapping", ErrInvalid, p.Name)
		}
		v.Set(p.Name, prune(section, p.Name, recorded))
		if err := v.UnmarshalKey(p.Name, o, strict); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrInvalid, p.Name, err)
		}
		if err := consistent(options, fs, previous, a, p.Name); err != nil {
			return nil, err
		}
		// The section of the file may state an answer, which takes the answer's value again. The
		// first call checked every answer, so this one returns nil.
		_ = answer(options, fs, a)
	}
	for _, f := range fs {
		m, ok := reflect.TypeAssert[interface{ Validate() error }](options.FieldByIndex(f.Index).Addr())
		if !ok {
			continue
		}
		if err := m.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %s.%s: %w", ErrInvalid, p.Name, f.Key, err)
		}
	}
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalid, p.Name, err)
	}
	return o, nil
}

// strict configures the decoder of viper for the options of a producer: keys by their yaml tags,
// inline structs by the option inline of the tag, an error for a key that the struct does not have,
// no conversion between the kinds of values and no hook, and a list or a map of the section in
// place of the one of the baseline. A field that the section lacks keeps its baseline value.
func strict(c *mapstructure.DecoderConfig) {
	c.TagName = configType
	c.SquashTagOption = inline
	c.ErrorUnused = true
	c.ZeroFields = true
	c.WeaklyTypedInput = false
	c.DecodeHook = nil
}

// answer sets each field of fs, the keys of the options v, that has an answer tag to the value of
// that answer of a. It returns an error that wraps [ErrDefect] for a field whose answer tag names
// no answer, and for a field whose type the answer does not convert to.
func answer(v reflect.Value, fs []Field, a *language.Answers) error {
	for _, f := range fs {
		if f.Answer == "" {
			continue
		}
		value, ok := answerOf(&f, a)
		if !ok {
			return fmt.Errorf("%w: the field %s of %s states the answer %q, which ergon init does not have as a %s",
				ErrDefect, f.Key, v.Type(), f.Answer, f.Type)
		}
		v.FieldByIndex(f.Index).Set(value)
	}
	return nil
}

// consistent returns an error that wraps [ErrInvalid] for the first field of fs, the keys of the
// options v of the section name, that has an answer tag and whose value, which the section states,
// is neither the answer of a nor the answer of previous. It returns nil for a nil previous. Each
// answer tag of fs names an answer that converts to its field, as [answer] checked.
func consistent(v reflect.Value, fs []Field, previous, a *language.Answers, name string) error {
	if previous == nil {
		return nil
	}
	for _, f := range fs {
		if f.Answer == "" {
			continue
		}
		stated := v.FieldByIndex(f.Index).Interface()
		current, _ := answerOf(&f, a)
		recorded, _ := answerOf(&f, previous)
		if reflect.DeepEqual(stated, current.Interface()) || reflect.DeepEqual(stated, recorded.Interface()) {
			continue
		}
		return fmt.Errorf("%w: %s.%s %q, which ergon init sync --%s sets, differs from the answer %q of ergon init",
			ErrInvalid, name, f.Key, fmt.Sprint(stated), f.Answer, fmt.Sprint(current.Interface()))
	}
	return nil
}

// answerOf returns the answer of a that the answer tag of f names, converted to the type of f. It
// reports false for a tag that names no answer, and for an answer that does not convert to the type
// of f.
func answerOf(f *Field, a *language.Answers) (reflect.Value, bool) {
	answers := reflect.ValueOf(a).Elem()
	i := slices.IndexFunc(reflect.VisibleFields(answers.Type()), func(g reflect.StructField) bool {
		name, _, _ := strings.Cut(g.Tag.Get("json"), ",")
		return name == f.Answer
	})
	if i < 0 || !answers.Field(i).Type().ConvertibleTo(f.Type) {
		return reflect.Value{}, false
	}
	return answers.Field(i).Convert(f.Type), true
}

// stated returns the value of v as .ergon.yaml states it: an empty list or map of the type of v for
// a nil list or map, which the writer writes as [] or {}, so the record states the value that the
// file reads back. It returns the value of v for any other value.
func stated(v reflect.Value) any {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	if v.Kind() == reflect.Map && v.IsNil() {
		return reflect.MakeMap(v.Type()).Interface()
	}
	return v.Interface()
}

// prune returns a copy of section, the section name of .ergon.yaml as viper parsed it, without each
// option of the section that recorded records with the value that section states. The decoder then
// keeps such an option at its baseline value. The decoder also does not see such an option when the
// producer no longer has it. The copy does not share a map with section.
func prune(section map[string]any, name string, recorded map[string]any) map[string]any {
	out := clone(section)
	for key, previous := range recorded {
		path, ok := strings.CutPrefix(key, name+".")
		if !ok {
			continue
		}
		parent := out
		keys := strings.Split(path, ".")
		for _, key := range keys[:len(keys)-1] {
			parent, _ = parent[key].(map[string]any)
		}
		last := keys[len(keys)-1]
		if value, ok := parent[last]; ok && same(value, previous) {
			delete(parent, last)
		}
	}
	return out
}

// clone returns a copy of m in which every map is a copy.
func clone(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if inner, ok := v.(map[string]any); ok {
			v = clone(inner)
		}
		out[k] = v
	}
	return out
}

// same reports whether a, a value that viper parsed, and b, a value that the lock records, have
// the same encoding in JSON, which orders the keys of a map. A value that JSON or YAML decodes,
// and the value of an option, encode without an error.
func same(a, b any) bool {
	x, err := json.Marshal(a)
	y, _ := json.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}
