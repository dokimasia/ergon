// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"bytes"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// lists is what a merge of two YAML documents does with a list that both have.
type lists uint8

const (
	// appendLists appends the items of the second list to the first, as a local file does.
	appendLists lists = 1

	// replaceLists replaces the first list with the second, as the keys of a configured file do.
	replaceLists lists = 2
)

// indent is the indentation of a YAML document that a merge encodes.
const indent = 2

// mergeYAML returns the YAML document base with the document over merged into it. Maps are merged
// key by key, and the value of over replaces the value of base for a key that both have, unless
// both values are maps or both are lists. Lists follow mode. A document with no content leaves
// the other one as it is. mergeYAML keeps the comments of both documents, and encodes a merged
// document with an indentation of two spaces. For a document that does not parse, it returns the
// error of the parser after "parse: ", which the caller wraps with the path of the file.
func mergeYAML(base, over []byte, mode lists) ([]byte, error) {
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
	mergeNode(dst.Content[0], src.Content[0], mode)
	return encodeYAML(&dst)
}

// mergeNode merges src into dst, as [mergeYAML] states.
func mergeNode(dst, src *yaml.Node, mode lists) {
	if dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(src.Content); i += 2 {
			key, value := src.Content[i], src.Content[i+1]
			j := mapIndex(dst, key.Value)
			if j < 0 {
				dst.Content = append(dst.Content, key, value)
				continue
			}
			mergeNode(dst.Content[j+1], value, mode)
		}
		return
	}
	if dst.Kind == yaml.SequenceNode && src.Kind == yaml.SequenceNode && mode == appendLists {
		dst.Content = append(dst.Content, src.Content...)
		return
	}
	*dst = *src
}

// mapIndex returns the index of the key named key in the map node m, or -1.
func mapIndex(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// encodeYAML returns the YAML document doc, indented by two spaces. A document that a parse
// returned, merged or not, encodes without an error, and the encoder writes to memory.
func encodeYAML(doc *yaml.Node) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(indent)
	err := errors.Join(enc.Encode(doc), enc.Close())
	return b.Bytes(), err
}
