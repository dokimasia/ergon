// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.yaml.in/yaml/v3"
)

// The layout of .ergon.yaml that [Write] encodes.
const (
	// indent is the indentation of a level of the document.
	indent = 2

	// width is the column after which the comment of a key wraps.
	width = 100
)

// Write returns the content of .ergon.yaml: existing, the current content, which is empty for a
// repository without the file, with the section of each of sections in place of the value of its
// key, or after the other keys, and without the sections of drop. It keeps every other key of
// existing, the comments of those keys, and the key of each section with its comments.
//
// Each section states every option of its producer, under the comment of the option's doc tag,
// with a list of scalars in flow style. viper parses the keys of .ergon.yaml in lowercase, so a key
// of a map that the file states in uppercase is written in lowercase. Write reports whether the
// result differs from existing apart from the layout of existing.
//
// It returns an error that wraps [ErrInvalid] for an existing that does not parse or whose root is
// not a mapping, and an error that wraps [ErrDefect] for options that are not a pointer to a struct
// or that YAML cannot encode.
func Write(existing []byte, sections []Section, drop []string) ([]byte, bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(existing, &doc); err != nil {
		return nil, false, fmt.Errorf("%w: parse: %w", ErrInvalid, err)
	}
	var normalized []byte
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	} else {
		if doc.Content[0].Kind != yaml.MappingNode {
			return nil, false, fmt.Errorf("%w: the document is not a mapping", ErrInvalid)
		}
		normalized = encode(&doc)
	}
	root := doc.Content[0]
	for _, name := range drop {
		if i := index(root, name); i >= 0 {
			root.Content = slices.Delete(root.Content, i, i+2)
		}
	}
	for _, s := range sections {
		value, err := section(s.Options)
		if err != nil {
			return nil, false, fmt.Errorf("%w: the section %s", err, s.Name)
		}
		if i := index(root, s.Name); i >= 0 {
			root.Content[i+1] = value
			continue
		}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.Name}, value)
	}
	out := encode(&doc)
	return out, !bytes.Equal(out, normalized), nil
}

// section returns the mapping of the options o, with the comment of each key and lists of scalars
// in flow style. It returns an error that wraps [ErrDefect] for options that are not a pointer to a
// struct, that declare a field wrong, or that YAML cannot encode.
func section(o language.Options) (*yaml.Node, error) {
	fs, err := Fields(o)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := n.Encode(o); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDefect, err)
	}
	for _, f := range fs {
		key, value := lookup(&n, f.Key)
		if key == nil {
			continue
		}
		depth := strings.Count(f.Key, ".") + 1
		key.HeadComment = comment(f.Doc, width-indent*depth-len("# "))
		if value.Kind == yaml.SequenceNode &&
			!slices.ContainsFunc(value.Content, func(e *yaml.Node) bool { return e.Kind != yaml.ScalarNode }) {

			value.Style = yaml.FlowStyle
		}
	}
	return &n, nil
}

// lookup returns the key node and the value node of the dotted key in the mapping n, and nil for
// a key that n lacks, as the encoding of a field with omitempty does.
func lookup(n *yaml.Node, key string) (k, v *yaml.Node) {
	v = n
	for name := range strings.SplitSeq(key, ".") {
		i := index(v, name)
		if i < 0 {
			return nil, nil
		}
		k, v = v.Content[i], v.Content[i+1]
	}
	return k, v
}

// comment returns doc wrapped at the word before the column columns, one line of the comment per
// line, which the encoder writes after "# ". It returns the empty string for an empty doc.
func comment(doc string, columns int) string {
	var b strings.Builder
	line := 0
	for word := range strings.FieldsSeq(doc) {
		switch {
		case line == 0:
		case line+1+len(word) > columns:
			b.WriteByte('\n')
			line = 0
		default:
			b.WriteByte(' ')
			line++
		}
		b.WriteString(word)
		line += len(word)
	}
	return b.String()
}

// index returns the index of the key named key in the mapping m, or -1.
func index(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// encode returns the YAML document doc, indented by two spaces. A document that a parse returned,
// or that holds nodes that an encode returned, encodes without an error, and the encoder writes to
// memory, so encode ignores the errors of the encoder.
func encode(doc *yaml.Node) []byte {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(indent)
	_ = enc.Encode(doc)
	_ = enc.Close()
	return b.Bytes()
}
