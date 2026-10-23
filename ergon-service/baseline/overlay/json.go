// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"slices"
)

// jsonIndent is the indentation of each level of a JSON document that a merge writes.
const jsonIndent = "  "

// node is a value of a JSON document that [MergeJSON] reads. An object keeps its members in their
// order, and an array keeps its items. A value of another kind is its token alone.
type node struct {
	// token is the begin delimiter of an object or an array, or the whole value of another kind. A
	// token of a string or a number keeps its text, with its escapes and its digits.
	token jsontext.Token

	// names are the names of the members of an object, in their order.
	names []jsontext.Token

	// items are the values of the members of an object, in the order of names, or the items of an
	// array.
	items []*node
}

// readJSON returns the value of the JSON document data, read token by token, and nil for a document
// without content. It returns an error after "parse: " for a document that is not valid JSON, with
// the error of the decoder, and for a second value after the value of the document.
func readJSON(data []byte) (*node, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	var root *node
	// open has the objects and the arrays that contain the next token, the innermost last.
	var open []*node
	for {
		end := dec.InputOffset()
		tok, err := dec.ReadToken()
		if errors.Is(err, io.EOF) {
			return root, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse: %w", err)
		}
		if root != nil && len(open) == 0 {
			return nil, fmt.Errorf("parse: a second value after offset %d", end)
		}
		tok = tok.Clone()
		kind := tok.Kind()
		if kind == '}' || kind == ']' {
			open = open[:len(open)-1]
			continue
		}
		if len(open) > 0 {
			parent := open[len(open)-1]
			// In an object, a name precedes each value.
			if parent.token.Kind() == '{' && len(parent.names) == len(parent.items) {
				parent.names = append(parent.names, tok)
				continue
			}
		}
		v := &node{token: tok}
		if len(open) == 0 {
			root = v
		} else {
			open[len(open)-1].items = append(open[len(open)-1].items, v)
		}
		if kind == '{' || kind == '[' {
			open = append(open, v)
		}
	}
}

// tokens appends the tokens of v to toks in the order of the text of v, and returns the extended
// slice.
func (v *node) tokens(toks []jsontext.Token) []jsontext.Token {
	toks = append(toks, v.token)
	switch v.token.Kind() {
	case '{':
		for i, name := range v.names {
			toks = append(toks, name)
			toks = v.items[i].tokens(toks)
		}
		return append(toks, jsontext.EndObject)
	case '[':
		for _, item := range v.items {
			toks = item.tokens(toks)
		}
		return append(toks, jsontext.EndArray)
	default:
		return toks
	}
}

// MergeJSON returns the JSON document base with the JSON document over merged into it. Objects are
// merged member by member: a member of over that base lacks follows the members of its object, and
// for a name that both have, the value of over replaces the value of base unless both values are
// objects, which merge, or both are arrays, to which the items of over are appended. A document
// without content leaves the other one as it is.
//
// MergeJSON writes the merged document as JSON.stringify(value, null, 2) of JavaScript writes it:
// each member and each item on a line of its own, an indentation of two spaces, and a newline at the
// end. It keeps the order of the members, and the text of each name, string and number.
//
// It returns an error after "parse: " for a document that is not one JSON value, such as a document
// with a comment, with a name twice in one object, or with a second value after its value.
func MergeJSON(base, over []byte) ([]byte, error) {
	dst, err := readJSON(base)
	if err != nil {
		return nil, err
	}
	src, err := readJSON(over)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return base, nil
	}
	if dst == nil {
		return over, nil
	}
	mergeNode(dst, src)
	var b bytes.Buffer
	enc := jsontext.NewEncoder(&b, jsontext.WithIndent(jsonIndent), jsontext.PreserveRawStrings(true))
	// The tokens of a document that a read returned, merged or not, follow the grammar of JSON, and the
	// encoder writes to memory, so the errors are nil.
	for _, tok := range dst.tokens(nil) {
		err = errors.Join(err, enc.WriteToken(tok))
	}
	return b.Bytes(), err
}

// mergeNode merges src into dst, as [MergeJSON] states.
func mergeNode(dst, src *node) {
	switch kind := dst.token.Kind(); {
	case kind == '{' && src.token.Kind() == '{':
		for i, name := range src.names {
			j := slices.IndexFunc(dst.names, func(n jsontext.Token) bool { return n.String() == name.String() })
			if j < 0 {
				dst.names = append(dst.names, name)
				dst.items = append(dst.items, src.items[i])
				continue
			}
			mergeNode(dst.items[j], src.items[i])
		}
	case kind == '[' && src.token.Kind() == '[':
		dst.items = append(dst.items, src.items...)
	default:
		*dst = *src
	}
}
