// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Set returns file with the value of each key of settings. file is the content of .ergon.yaml, and a
// key is dotted, such as license.parameters.change-date. Set writes each value as a string. It adds
// each mapping of a key that file does not have, and keeps every other key of file with its
// comments. Set returns file for no settings.
//
// It returns an error that wraps [ErrInvalid] for a file that does not parse or whose root is not a
// mapping. It also returns such an error for a key whose path crosses a value that is not a mapping.
// The text of that error contains the dotted path of the value.
func Set(file []byte, settings map[string]string) ([]byte, error) {
	if len(settings) == 0 {
		return file, nil
	}
	doc, _, err := parse(file)
	if err != nil {
		return nil, err
	}
	for _, key := range slices.Sorted(maps.Keys(settings)) {
		m := doc.Content[0]
		names := strings.Split(key, ".")
		last := len(names) - 1
		for i, name := range names[:last] {
			j := index(m, name)
			if j < 0 {
				m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: strTag, Value: name},
					&yaml.Node{Kind: yaml.MappingNode, Tag: mapTag})
				j = len(m.Content) - 2
			}
			if m = m.Content[j+1]; m.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("%w: %s, which is not a mapping", ErrInvalid, strings.Join(names[:i+1], "."))
			}
		}
		value := &yaml.Node{Kind: yaml.ScalarNode, Tag: strTag, Value: settings[key]}
		if j := index(m, names[last]); j >= 0 {
			m.Content[j+1] = value
			continue
		}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: strTag, Value: names[last]}, value)
	}
	return encode(doc), nil
}
