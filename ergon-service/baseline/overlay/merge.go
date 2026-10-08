// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay

import (
	"bytes"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// indent is the indentation of a YAML document that a merge encodes.
const indent = 2

// Merge returns the YAML document base with the document over merged into it. Maps are merged key
// by key: a key of over that base lacks is appended to the map, and for a key that both have, the
// value of over replaces the value of base unless both values are maps, which merge, or both are
// lists, to which the items of over are appended. A document without content leaves the other one
// as it is. Merge keeps the comments of both documents, and encodes a merged document with an
// indentation of two spaces. For a document that does not parse, it returns the error of the
// parser after "parse: ".
func Merge(base, over []byte) ([]byte, error) {
	var dst, src yaml.Node
	if err := yaml.Unmarshal(base, &dst); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if err := yaml.Unmarshal(over, &src); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(src.Content) == 0 {
		return base, nil
	}
	if len(dst.Content) == 0 {
		return over, nil
	}
	merge(dst.Content[0], src.Content[0])
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(indent)
	// A document that a parse returned, merged or not, encodes without an error, and the encoder
	// writes to memory, so the errors are nil.
	err := errors.Join(enc.Encode(&dst), enc.Close())
	return b.Bytes(), err
}

// merge merges src into dst, as [Merge] states.
func merge(dst, src *yaml.Node) {
	if dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode {
		// The content of a map node has a key and a value for each entry, so its length is even.
		for i := 0; i < len(src.Content); i += 2 {
			key, value := src.Content[i], src.Content[i+1]
			j := index(dst, key.Value)
			if j < 0 {
				dst.Content = append(dst.Content, key, value)
				continue
			}
			merge(dst.Content[j+1], value)
		}
		return
	}
	if dst.Kind == yaml.SequenceNode && src.Kind == yaml.SequenceNode {
		dst.Content = append(dst.Content, src.Content...)
		return
	}
	*dst = *src
}

// index returns the index of the key named key in the map node m, or -1.
func index(m *yaml.Node, key string) int {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}
