// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/pelletier/go-toml/v2"
)

// MergeTOML returns the TOML document base with the TOML document over merged into it. over is a
// valid TOML document of its own. Tables are merged key by key: a key of over that base lacks is
// added to its table, and for a key that both have, the value of over replaces the value of base
// unless both values are arrays, to which the items of over are appended, or both are inline
// tables, which merge in the same way. A table of over that base lacks follows the tables of base.
// A table of an array of tables of over, with the tables inside it, follows the last table of that
// array in base. A document without content leaves the other one as it is.
//
// MergeTOML keeps the comments of both documents, except the comment at the end of the line of a
// value of over that merges into a value of base. A key that merges keeps the comments above it in
// base, followed by the comments above it in over.
//
// It returns an error after "parse: " for a document that does not parse, after "invalid: " for an
// over that is not valid TOML, such as one that defines a key twice, and after "conflict: " for a
// merged document that is not valid TOML, such as one with a table that over adds and base defines
// inline.
func MergeTOML(base, over []byte) ([]byte, error) {
	dst, err := tomledit.Parse(bytes.NewReader(base))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	src, err := tomledit.Parse(bytes.NewReader(over))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	// The parser of tomledit leaves the rules of TOML unchecked, such as a key defined twice, so a
	// decoder checks over.
	var tree map[string]any
	if err = toml.Unmarshal(over, &tree); err != nil {
		return nil, fmt.Errorf("invalid: %w", err)
	}
	if len(src.Global.Items) == 0 && len(src.Sections) == 0 {
		return base, nil
	}
	if len(dst.Global.Items) == 0 && len(dst.Sections) == 0 {
		return over, nil
	}
	mergeTable(dst, dst.Global, src.Global)
	for i := 0; i < len(src.Sections); i++ {
		s := src.Sections[i]
		if s.IsArray {
			// The table of an array of tables brings the tables inside it, which follow it up to the
			// next table of the array.
			j := i + 1
			for j < len(src.Sections) && s.Name.IsPrefixOf(src.Sections[j].Name) &&
				len(s.Name) < len(src.Sections[j].Name) {
				j++
			}
			dst.Sections = slices.Insert(dst.Sections, afterArray(dst.Sections, s.Name), src.Sections[i:j]...)
			i = j - 1
			continue
		}
		k := slices.IndexFunc(
			dst.Sections,
			func(t *tomledit.Section) bool { return !t.IsArray && t.Name.Equals(s.Name) },
		)
		if k >= 0 {
			mergeTable(dst, dst.Sections[k], s)
			continue
		}
		added := &tomledit.Section{Heading: s.Heading}
		mergeTable(dst, added, s)
		// A table whose keys all merged into the dotted keys of base does not add its name, and an
		// empty table adds its name.
		if len(added.Items) > 0 || len(s.Items) == 0 {
			dst.Sections = append(dst.Sections, added)
		}
	}
	var b bytes.Buffer
	err = tomledit.Format(&b, dst)
	var merged map[string]any
	if err = errors.Join(err, toml.Unmarshal(b.Bytes(), &merged)); err != nil {
		return nil, fmt.Errorf("conflict: %w", err)
	}
	return b.Bytes(), nil
}

// mergeTable merges the items of the section from of over into the section into, which belongs to
// the document base or follows it, in their order. A key that base has outside an array of tables
// merges with [mergeValue]. Every other key and every block of comments is appended to into.
func mergeTable(base *tomledit.Document, into, from *tomledit.Section) {
	for _, item := range from.Items {
		if kv, ok := item.(*parser.KeyValue); ok {
			e := base.First(slices.Concat(from.TableName(), kv.Name)...)
			if e != nil && e.IsMapping() && (e.IsGlobal() || !e.IsArray) {
				mergeValue(e.KeyValue, kv)
				continue
			}
		}
		into.Items = append(into.Items, item)
	}
}

// mergeValue merges the value of src into the value of dst, as [MergeTOML] states: it appends the
// items of two arrays, merges two inline tables key by key, and otherwise takes the value of src. The
// comments above src follow the comments above dst.
func mergeValue(dst, src *parser.KeyValue) {
	dst.Block = append(dst.Block, src.Block...)
	switch d := dst.Value.X.(type) {
	case parser.Array:
		if s, ok := src.Value.X.(parser.Array); ok {
			d = append(d, s...)
			dst.Value.X = d
			return
		}
	case parser.Inline:
		if s, ok := src.Value.X.(parser.Inline); ok {
			for _, kv := range s {
				i := slices.IndexFunc(d, func(t *parser.KeyValue) bool { return t.Name.Equals(kv.Name) })
				if i < 0 {
					d = append(d, kv)
					continue
				}
				mergeValue(d[i], kv)
			}
			dst.Value.X = d
			return
		}
	}
	dst.Value = src.Value
}

// afterArray returns the index in sections after the last table of the array of tables name and
// after the tables inside that table, and the length of sections when sections has no table of that
// array.
func afterArray(sections []*tomledit.Section, name parser.Key) int {
	last := -1
	for i, s := range sections {
		if s.IsArray && s.Name.Equals(name) {
			last = i
		}
	}
	if last < 0 {
		return len(sections)
	}
	i := last + 1
	for i < len(sections) && name.IsPrefixOf(sections[i].Name) && len(name) < len(sections[i].Name) {
		i++
	}
	return i
}
