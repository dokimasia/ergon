// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
)

// inline is the option of a yaml tag that puts the keys of a struct field into the mapping of its
// parent.
const inline = "inline"

// stateless are the kinds of a value that no document of YAML states. The encoder of YAML panics
// for them.
var stateless = []reflect.Kind{reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128}

// Field is a key of the options of a producer. An exported field states the key in the options'
// struct, in a group of it, or in a struct that one of them embeds inline.
type Field struct {
	// Type is the type of the field. A field whose type is a struct is a group of options.
	Type reflect.Type

	// Key is the key of the field below the section, with the keys of its groups before it,
	// separated by dots, such as fuzz.time.
	Key string

	// Doc is the meaning of the field, which ergon init writes as the comment above its key, or
	// empty.
	Doc string

	// Answer is the key of the answer of ergon init that the field states, or empty.
	Answer string

	// Index is the index sequence of the field from the options' struct, through every group and
	// every inline struct, as reflect.Value.FieldByIndex takes it.
	Index []int
}

// Fields returns every key of the options o at every depth, in the order of the fields: a group
// before its keys, and the keys of an inline struct in place of the struct. It skips a field tagged
// yaml:"-" and an unexported field that its struct does not embed. It returns an error that wraps
// [ErrDefect] for options that are not a pointer to a struct, for an exported field without a key,
// for an inline field that is not a struct, for an embedded field without the option inline, which
// YAML and viper read in different ways, and for a field of a kind that no document of YAML states,
// such as a function.
func Fields(o language.Options) ([]Field, error) {
	t := reflect.TypeOf(o)
	if t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: options of the type %v, which is not a pointer to a struct", ErrDefect, t)
	}
	return fields(t.Elem(), "", nil)
}

// fields returns every key of the struct type t at every depth, as [Fields] states. prefix is the
// key of t, with a dot, or empty for the options' struct, and index is the index sequence of t.
func fields(t reflect.Type, prefix string, index []int) ([]Field, error) {
	var out []Field
	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get(configType)
		if tag == "-" || (!f.IsExported() && !f.Anonymous) {
			continue
		}
		at := append(slices.Clone(index), i)
		name, opts, _ := strings.Cut(tag, ",")
		if slices.Contains(strings.Split(opts, ","), inline) {
			if f.Type.Kind() != reflect.Struct {
				return nil, fmt.Errorf("%w: the inline field %s of %s is not a struct", ErrDefect, f.Name, t)
			}
			inner, err := fields(f.Type, prefix, at)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
			continue
		}
		if f.Anonymous {
			return nil, fmt.Errorf("%w: the embedded field %s of %s has no yaml tag with the option inline", ErrDefect,
				f.Name, t)
		}
		if name == "" {
			return nil, fmt.Errorf("%w: the field %s of %s has no yaml key", ErrDefect, f.Name, t)
		}
		if slices.Contains(stateless, f.Type.Kind()) {
			return nil, fmt.Errorf("%w: the field %s of %s is a %s, which no document of YAML states", ErrDefect,
				f.Name, t, f.Type.Kind())
		}
		out = append(out, Field{
			Type:   f.Type,
			Key:    prefix + name,
			Doc:    f.Tag.Get(option.DocTag),
			Answer: f.Tag.Get(option.AnswerTag),
			Index:  at,
		})
		if f.Type.Kind() == reflect.Struct {
			inner, err := fields(f.Type, prefix+name+".", at)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
		}
	}
	return out, nil
}
