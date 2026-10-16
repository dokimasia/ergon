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

	"go.dokimi.dev/ergon/core/language"
	"go.stealthscale.io/cli"
	"go.yaml.in/yaml/v3"
)

// ErrInvalid is the error for .ergon.yaml that does not parse, a section with a key that its
// producer's struct does not have or a value that does not decode into its field, a value that its
// option does not accept, and the section of a producer that the repository does not have. Its
// text names the key.
var ErrInvalid = errors.New("options: invalid .ergon.yaml")

// ErrDefect is the error for options that a producer declares wrong, which is a defect of the
// producer:
//
//   - options that are not a pointer to a struct
//   - a field without a yaml key, and an inline field that is not a struct
//   - a field whose answer tag is not an answer of ergon init, or whose type the answer does not
//     convert to
//   - a field that go-cli does not decode, such as a field whose name tag is not in lower case
var ErrDefect = errors.New("options: invalid options of a producer")

// nullTag is the tag of a null of YAML, such as ~ or a key without a value.
const nullTag = "!!null"

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
// lock, which a may change, and nil for a repository without a lock. yaml.v3 parses file, and
// go-cli decodes each section into a new value of its producer's options, which starts with the
// baseline value of every option that the section lacks. A scalar of the section converts from its
// text as file writes it, a key of a map keeps its case, and a null keeps the baseline value. An
// option whose value in file equals the value that recorded records for it takes its baseline
// value, so an option that the repository never changed follows the baseline of the installed
// ergon. Resolve leaves out such an option that the producer no longer has, and a group of such
// options alone, so the baseline of the installed ergon removes them. A field with an answer tag
// takes the answer's value of a.
// names are the names of every section that a producer of the catalog can have: a section of names
// that no producer of producers states is dropped when recorded records an option of it.
//
// A section that states an answer with the value of a or of previous, which ergon init wrote before
// a changed it, is consistent with the lock. ergon init writes the answers of a repository without
// a lock from a alone, so for a nil previous a section may state any value of an answer.
//
// It returns an error that wraps [ErrInvalid] and contains the key for each of these:
//
//   - a file that does not parse, and a section that is not a mapping
//   - a mapping with a key twice
//   - a key that a section does not have, and a value that does not decode into its field
//   - an answer that a section states with a value that is neither the answer of a nor of a
//     non-nil previous
//   - a value whose Validate method returns an error, and an error of the Validate of a producer's
//     options
//   - a section of names whose producer the repository does not have, when recorded does not
//     record an option of the section
//
// The error of a key or of a value also wraps the error of go-cli, such as [cli.ErrUnknownKey].
// Resolve returns an error that wraps [ErrDefect] for options that a producer declares wrong. It
// reads file, recorded, previous and a, and does not modify them.
func Resolve(file []byte, recorded map[string]any, previous, a *language.Answers, producers []Producer,
	names []string,
) (Resolution, error) {
	doc, _, err := parse(file)
	if err != nil {
		return Resolution{}, err
	}
	root := doc.Content[0]
	res := Resolution{Record: map[string]any{}}
	for _, p := range producers {
		o, err := resolve(root, &p, recorded, previous, a, res.Record)
		if err != nil {
			return Resolution{}, err
		}
		res.Sections = append(res.Sections, Section{Name: p.Name, Options: o})
	}
	keys := slices.Collect(maps.Keys(recorded))
	for _, name := range names {
		own := slices.ContainsFunc(producers, func(p Producer) bool { return p.Name == name })
		if own || member(root, name) == nil {
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
// options to record. root is the mapping of the document of .ergon.yaml. resolve removes from it
// each option of the section of p that equals its record.
func resolve(root *yaml.Node, p *Producer, recorded map[string]any, previous, a *language.Answers,
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
	tree := map[string]any{}
	section := member(root, p.Name)
	if section != nil {
		if section.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%w: %s, which must be a mapping", ErrInvalid, p.Name)
		}
		for key, value := range recorded {
			if path, ok := strings.CutPrefix(key, p.Name+"."); ok {
				prune(section, strings.Split(path, "."), value)
			}
		}
		var t text
		if err := section.Decode(&t); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrInvalid, p.Name, err)
		}
		tree[p.Name] = t.value
	}
	// cli.Decode declares the types of o before it looks up the section, so it also reports a
	// defect of o for a file without the section.
	if err := cli.Decode(tree, p.Name, o); err != nil {
		if errors.Is(err, cli.ErrDeclaration) {
			return nil, fmt.Errorf("%w: the section %s: %w", ErrDefect, p.Name, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if section != nil {
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

// member returns the value of the key name in the mapping m, or nil for a key that m does not have
// and for a null, such as a key without a value.
func member(m *yaml.Node, name string) *yaml.Node {
	i := index(m, name)
	if i < 0 || m.Content[i+1].ShortTag() == nullTag {
		return nil
	}
	return m.Content[i+1]
}

// prune removes the option at keys from the mapping m of a section when its value equals previous,
// the value that the lock records for it, as [same] compares them. keys are the names of the
// groups of the option, and its own name last. prune also removes each group that the removal
// leaves empty. It reports whether it removed the option. The decoder keeps a removed option at its
// baseline value, and it does not see a removed option or group that the producer no longer has.
func prune(m *yaml.Node, keys []string, previous any) bool {
	i := index(m, keys[0])
	if i < 0 {
		return false
	}
	value := m.Content[i+1]
	if len(keys) == 1 {
		var v any
		if err := value.Decode(&v); err != nil || !same(v, previous) {
			return false
		}
	} else if value.Kind != yaml.MappingNode || !prune(value, keys[1:], previous) {
		return false
	} else if len(value.Content) > 0 {
		return true
	}
	m.Content = slices.Delete(m.Content, i, i+2)
	return true
}

// same reports whether a, a value that yaml.v3 decodes, and b, a value that the lock records, have
// the same encoding in JSON, which orders the keys of a map. A value that JSON decodes, and the
// value of an option, encode without an error.
func same(a, b any) bool {
	x, err := json.Marshal(a)
	y, _ := json.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}

// text is a member of .ergon.yaml in the tree that go-cli decodes: a map[string]any for a mapping,
// a []any for a sequence, nil for a null, and the text of any other scalar as the file writes it.
// An option that is a string receives an unquoted 1.20 as 1.20.
type text struct {
	value any
}

// UnmarshalYAML sets t to the member n. yaml.v3 resolves an alias and the merge keys of a mapping
// before it calls the method. It does not call the method for a null, so the value of a null is
// nil. UnmarshalYAML decodes each element of a sequence on its own, because yaml.v3 leaves out a
// null element when it decodes a sequence into a slice of text. It returns the error of yaml.v3 for
// a mapping with a key twice.
func (t *text) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.MappingNode:
		var members map[string]text
		if err := n.Decode(&members); err != nil {
			return err //nolint:wrapcheck // yaml.v3 collects a *yaml.TypeError that it receives unwrapped.
		}
		m := make(map[string]any, len(members))
		for k, v := range members {
			m[k] = v.value
		}
		t.value = m
	case yaml.SequenceNode:
		list := make([]any, len(n.Content))
		for i, e := range n.Content {
			var item text
			if err := e.Decode(&item); err != nil {
				return err //nolint:wrapcheck // yaml.v3 collects a *yaml.TypeError that it receives unwrapped.
			}
			list[i] = item.value
		}
		t.value = list
	default:
		t.value = n.Value
	}
	return nil
}
