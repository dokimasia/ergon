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

// The tags of the nodes that [Write] and [Set] add to a document.
const (
	// mapTag is the tag of a mapping.
	mapTag = "!!map"

	// strTag is the tag of a string. The encoder quotes a string that YAML would read as another
	// type, such as 2030-01-01.
	strTag = "!!str"
)

// Write returns the content of .ergon.yaml with sections and without the sections of drop. existing
// is the current content. It is empty for a repository without the file. Each section replaces the
// value of its key, or follows the other keys when existing does not have the key. Write keeps every
// other key of existing with its comments, and the key of each section with its comments.
//
// Write writes each option of a section under a comment with the doc tag of the option. It writes a
// list of scalars in flow style. viper parses the keys of .ergon.yaml in lowercase, so a section has
// the keys of its maps in lowercase. Write reports whether the result differs from existing in more
// than the layout of existing.
//
// It returns an error that wraps [ErrInvalid] for an existing that does not parse or whose root is
// not a mapping. It returns an error that wraps [ErrDefect] for options that are not a pointer to a
// struct or that YAML cannot encode.
func Write(existing []byte, sections []Section, drop []string) ([]byte, bool, error) {
	doc, rooted, err := parse(existing)
	if err != nil {
		return nil, false, err
	}
	var normalized []byte
	if rooted {
		normalized = encode(doc)
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
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: strTag, Value: s.Name}, value)
	}
	out := encode(doc)
	return out, !bytes.Equal(out, normalized), nil
}

// parse returns the document of file and reports whether file has a root. file is the content of
// .ergon.yaml. The root of the document is a mapping, and it is empty for a file without a root. It
// returns an error that wraps [ErrInvalid] for a file that does not parse or whose root is not a
// mapping.
func parse(file []byte) (*yaml.Node, bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(file, &doc); err != nil {
		return nil, false, fmt.Errorf("%w: parse: %w", ErrInvalid, err)
	}
	if len(doc.Content) == 0 {
		return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: mapTag}}},
			false, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("%w: the document is not a mapping", ErrInvalid)
	}
	return &doc, true, nil
}

// section returns the mapping of the options o. Each key has the comment of its doc tag, and a list
// of scalars is in flow style. It returns an error that wraps [ErrDefect] for options that are not a
// pointer to a struct, that declare a field wrong, or that YAML cannot encode.
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

// lookup returns the key node and the value node of the dotted key in the mapping n. It returns nil
// for a key that n does not have, such as a field with omitempty that the encoder left out.
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

// comment returns the words of doc in lines of at most columns characters. A word longer than a
// line has a line of its own. The encoder writes each line after "# ". comment returns the
// empty string for an empty doc.
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

// encode returns the YAML document doc with an indentation of two spaces. It ignores the errors of
// the encoder. The encoder writes to memory, and it encodes every node of a parse or an encode and
// every node that Set adds.
func encode(doc *yaml.Node) []byte {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(indent)
	_ = enc.Encode(doc)
	_ = enc.Close()
	return b.Bytes()
}
