// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"

	"go.dokimi.dev/ergon/core/workspace"
)

// ManifestFile is the manifest of a package, in the directory of the package.
const ManifestFile = "package.json"

// The keys of package.json that a release reads.
const (
	keyName       = "name"
	keyVersion    = "version"
	keyPrivate    = "private"
	keyWorkspaces = "workspaces"
	keyPackages   = "packages"
)

// topLevel is the name of the top-level value of a package.json in the errors.
const topLevel = "the top-level value"

// bom is the byte order mark of UTF-8. npm skips it at the start of a package.json.
const bom = "\xef\xbb\xbf"

// whitespace is the whitespace that JSON allows around a value.
const whitespace = " \t\n\r"

// sections are the keys of package.json that list the requirements of a package, with the kind of
// each.
var sections = map[string]workspace.Kind{
	"dependencies":         workspace.KindRuntime,
	"optionalDependencies": workspace.KindOptional,
	"peerDependencies":     workspace.KindPeer,
	"devDependencies":      workspace.KindDev,
}

// Span is the range of bytes of a value in a file, from Start to End.
type Span struct {
	Start, End int
}

// Entry is a requirement of a package in its package.json.
type Entry struct {
	// Name is the name of the required package, the key of the requirement.
	Name string

	// Text is the requirement as the string of the file states it, such as ^1.2.0 or workspace:*.
	Text string

	// Kind is the section of the requirement: [workspace.KindRuntime] for dependencies,
	// [workspace.KindOptional] for optionalDependencies, [workspace.KindPeer] for
	// peerDependencies, and [workspace.KindDev] for devDependencies.
	Kind workspace.Kind

	// Span is the literal of the string of the requirement in the file, with its quotes.
	Span Span
}

// Manifest is the content of a package.json that a release reads.
type Manifest struct {
	// Name is the name of the package, and empty for a file without one.
	Name string

	// Version is the version of the package as the file writes it, and empty for a file without
	// one.
	Version string

	// Workspaces are the globs of the packages of an npm workspace, from workspaces or from
	// workspaces.packages.
	Workspaces []string

	// Requirements are the requirements of the package, in the order of the file.
	Requirements []Entry

	// VersionSpan is the literal of the string of the version in the file, with its quotes.
	VersionSpan Span

	// Private reports a package that a registry does not receive, whose private is true.
	Private bool
}

// parser reads a package.json into a manifest.
type parser struct {
	// d decodes data after its byte order mark.
	d *jsontext.Decoder

	// m is the manifest that the parser fills.
	m *Manifest

	// data is the content of the file.
	data []byte

	// offset is the length of the byte order mark at the start of data. The offsets of d start
	// after it.
	offset int
}

// ParseManifest returns the manifest of the package.json data. file is the name of the file in the
// errors. ParseManifest reads the members of the object of the file that a release reads, and skips
// every other member. It skips a byte order mark at the start of data, as npm does.
//
// It returns an error that wraps [ErrManifest] and contains the name of the file for these inputs:
//
//   - data that is not a JSON object, or data after the object
//   - a member that the object repeats
//   - a name, a version, a requirement or a glob that is not a string
//   - a private that is not a boolean
//   - a section of requirements that is not an object
//   - workspaces that are neither a list nor an object
func ParseManifest(file string, data []byte) (*Manifest, error) {
	body := bytes.TrimPrefix(data, []byte(bom))
	p := &parser{
		d:      jsontext.NewDecoder(bytes.NewReader(body)),
		m:      &Manifest{},
		data:   data,
		offset: len(data) - len(body),
	}
	if err := p.object(topLevel, p.member); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	if rest := bytes.TrimLeft(body[p.d.InputOffset():], whitespace); len(rest) > 0 {
		return nil, fmt.Errorf("%w: %s, which has data after its object", ErrManifest, file)
	}
	return p.m, nil
}

// object reads the object at key, and calls member with the name of each of its members in the
// order of the file. member reads the value of the member. object returns the error of the decoder
// and of member, and an error for a value that is not an object.
func (p *parser) object(key string, member func(name string) error) error {
	tok, err := p.d.ReadToken()
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if tok.Kind() != jsontext.KindBeginObject {
		return fmt.Errorf("%s, which is not an object", key)
	}
	for {
		tok, err := p.d.ReadToken()
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if tok.Kind() == jsontext.KindEndObject {
			return nil
		}
		// The next call of the decoder voids tok, so member receives a copy of its name.
		if err := member(tok.String()); err != nil {
			return err
		}
	}
}

// member reads the value of the member name of the object of the file into the manifest, and skips
// the value of a member that a release does not read. It returns the error of the decoder, and an
// error for a value whose kind differs from the kind of its member.
func (p *parser) member(name string) error {
	var err error
	switch name {
	case keyName:
		p.m.Name, _, err = p.text(name)
	case keyVersion:
		p.m.Version, p.m.VersionSpan, err = p.text(name)
	case keyPrivate:
		p.m.Private, err = p.boolean(name)
	case keyWorkspaces:
		p.m.Workspaces, err = p.workspaces()
	default:
		kind, ok := sections[name]
		if !ok {
			if err = p.d.SkipValue(); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			return nil
		}
		err = p.object(name, func(dep string) error { return p.entry(name+"."+dep, dep, kind) })
	}
	return err
}

// entry reads the requirement at key on the package dep, of the section kind, into the manifest.
// It returns the error of the decoder, and an error for a requirement that is not a string.
func (p *parser) entry(key, dep string, kind workspace.Kind) error {
	text, span, err := p.text(key)
	if err != nil {
		return err
	}
	p.m.Requirements = append(p.m.Requirements, Entry{Name: dep, Text: text, Kind: kind, Span: span})
	return nil
}

// workspaces reads the value of workspaces: a list of globs, or an object whose member packages is a
// list of globs. It returns the error of the decoder, and an error for a value of another kind.
func (p *parser) workspaces() ([]string, error) {
	switch p.d.PeekKind() {
	case jsontext.KindBeginArray:
		return p.globs(keyWorkspaces)
	case jsontext.KindBeginObject:
		var globs []string
		err := p.object(keyWorkspaces, func(name string) error {
			if name != keyPackages {
				if err := p.d.SkipValue(); err != nil {
					return fmt.Errorf("%s.%s: %w", keyWorkspaces, name, err)
				}
				return nil
			}
			list, err := p.globs(keyWorkspaces + "." + keyPackages)
			globs = list
			return err
		})
		return globs, err
	default:
		if _, err := p.d.ReadToken(); err != nil {
			return nil, fmt.Errorf("%s: %w", keyWorkspaces, err)
		}
		return nil, fmt.Errorf("%s, which is neither a list nor an object", keyWorkspaces)
	}
}

// globs reads the list at key, whose items are strings. It returns the error of the decoder, and an
// error for a value that is not a list and for an item that is not a string.
func (p *parser) globs(key string) ([]string, error) {
	tok, err := p.d.ReadToken()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if tok.Kind() != jsontext.KindBeginArray {
		return nil, fmt.Errorf("%s, which is not a list", key)
	}
	var globs []string
	for {
		tok, err := p.d.ReadToken()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		switch tok.Kind() {
		case jsontext.KindEndArray:
			return globs, nil
		case jsontext.KindString:
			globs = append(globs, tok.String())
		default:
			return nil, fmt.Errorf("%s, which has an item that is not a string", key)
		}
	}
}

// text reads the string at key, and returns its text with the span of its literal in the file. The
// literal starts at the first quote after the token before it, because only whitespace, a colon and
// a comma separate two tokens. It returns the error of the decoder, and an error for a value that is
// not a string.
func (p *parser) text(key string) (string, Span, error) {
	start := p.offset + int(p.d.InputOffset())
	tok, err := p.d.ReadToken()
	if err != nil {
		return "", Span{}, fmt.Errorf("%s: %w", key, err)
	}
	if tok.Kind() != jsontext.KindString {
		return "", Span{}, fmt.Errorf("%s, which is not a string", key)
	}
	end := p.offset + int(p.d.InputOffset())
	start += bytes.IndexByte(p.data[start:end], '"')
	return tok.String(), Span{Start: start, End: end}, nil
}

// boolean reads the boolean at key. It returns the error of the decoder, and an error for a value
// that is not a boolean.
func (p *parser) boolean(key string) (bool, error) {
	tok, err := p.d.ReadToken()
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	if kind := tok.Kind(); kind != jsontext.KindTrue && kind != jsontext.KindFalse {
		return false, fmt.Errorf("%s, which is not a boolean", key)
	}
	return tok.Bool(), nil
}
