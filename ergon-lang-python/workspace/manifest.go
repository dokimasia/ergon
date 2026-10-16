// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2/unstable"
	"go.dokimi.dev/ergon/core/workspace"
)

// The keys of pyproject.toml that a release reads.
const (
	keyProject      = "project"
	keyName         = "name"
	keyVersion      = "version"
	keyDynamic      = "dynamic"
	keyClassifiers  = "classifiers"
	keyDependencies = "dependencies"
	keyOptional     = "optional-dependencies"
	keyGroups       = "dependency-groups"
	keyTool         = "tool"
	keyUV           = "uv"
	keyPackage      = "package"
	keyDev          = "dev-dependencies"
	keyWorkspace    = "workspace"
	keyMembers      = "members"
	keyExclude      = "exclude"
)

// privateClassifier is the classifier of a project that PyPI refuses to receive.
const privateClassifier = "Private :: Do Not Upload"

// falseValue is the boolean false of TOML.
const falseValue = "false"

// The paths of the keys of pyproject.toml that a release reads.
var (
	pathName         = []string{keyProject, keyName}
	pathVersion      = []string{keyProject, keyVersion}
	pathDynamic      = []string{keyProject, keyDynamic}
	pathClassifiers  = []string{keyProject, keyClassifiers}
	pathDependencies = []string{keyProject, keyDependencies}
	pathOptional     = []string{keyProject, keyOptional}
	pathDev          = []string{keyTool, keyUV, keyDev}
	pathPackage      = []string{keyTool, keyUV, keyPackage}
	pathWorkspace    = []string{keyTool, keyUV, keyWorkspace}
	pathMembers      = []string{keyTool, keyUV, keyWorkspace, keyMembers}
	pathExclude      = []string{keyTool, keyUV, keyWorkspace, keyExclude}
)

// Span is the range of bytes of a value in a file, from Start to End.
type Span struct {
	Start, End int
}

// Entry is a requirement of a project in its pyproject.toml.
type Entry struct {
	// Text is the requirement as the string of the file states it.
	Text string

	// Kind is the section of the requirement: [workspace.KindRuntime] for project.dependencies,
	// [workspace.KindOptional] for project.optional-dependencies, and [workspace.KindDev] for a
	// dependency group and for tool.uv.dev-dependencies.
	Kind workspace.Kind

	Requirement

	// Span is the literal of the string in the file, with its quotes.
	Span Span
}

// Manifest is the content of a pyproject.toml that a release reads.
type Manifest struct {
	// Name is project.name as the file writes it.
	Name string

	// Version is project.version as the file writes it, and empty for a file without one.
	Version string

	// Requirements are the requirements of the project, in the order of the file.
	Requirements []Entry

	// Members and Exclude are the globs of tool.uv.workspace.
	Members, Exclude []string

	// VersionSpan is the literal of project.version in the file, with its quotes.
	VersionSpan Span

	// Project reports a table project.
	Project bool

	// Dynamic reports that project.dynamic lists version, so a build reads the version from
	// elsewhere, such as the tags of git.
	Dynamic bool

	// Private reports a project that a registry does not receive: tool.uv.package is false, or a
	// classifier is Private :: Do Not Upload.
	Private bool

	// Workspace reports a table tool.uv.workspace, so the project is the root of a uv workspace.
	Workspace bool
}

// ParseManifest returns the manifest of the pyproject.toml data. file is the name of the file in
// the errors. ParseManifest reads the keys of its tables, of their dotted keys and of their inline
// tables, and skips each array of tables.
//
// It returns an error that wraps [ErrManifest] and names the file and the key for data that does
// not parse, for a value of the wrong type, and for a requirement that does not parse.
func ParseManifest(file string, data []byte) (*Manifest, error) {
	m := &Manifest{}
	var p unstable.Parser
	p.Reset(data)
	var table []string
	array := false
	for p.NextExpression() {
		// The parser keeps no comment, so an expression is a key, a table or an array of tables.
		e := p.Expression()
		if e.Kind != unstable.KeyValue {
			table, array = keys(e.Key()), e.Kind == unstable.ArrayTable
			m.mark(table)
			continue
		}
		if array {
			continue
		}
		if err := m.read(file, slices.Concat(table, keys(e.Key())), e.Value()); err != nil {
			return nil, err
		}
	}
	if err := p.Error(); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	return m, nil
}

// read reads the value at the key path of file into m, and the value of each key of an inline
// table. It returns an error that wraps [ErrManifest] for a value of the wrong type and for a
// requirement that does not parse.
func (m *Manifest) read(file string, path []string, value *unstable.Node) error {
	m.mark(path)
	if value.Kind == unstable.InlineTable {
		it := value.Children()
		for it.Next() {
			kv := it.Node()
			if err := m.read(file, slices.Concat(path, keys(kv.Key())), kv.Value()); err != nil {
				return err
			}
		}
		return nil
	}
	var err error
	switch key := strings.Join(path, "."); {
	case slices.Equal(path, pathName):
		m.Name, err = text(file, key, value)
	case slices.Equal(path, pathVersion):
		m.Version, err = text(file, key, value)
		m.VersionSpan = span(value)
	case slices.Equal(path, pathDynamic):
		var list []string
		list, err = texts(file, key, value)
		m.Dynamic = slices.Contains(list, keyVersion)
	case slices.Equal(path, pathClassifiers):
		var list []string
		list, err = texts(file, key, value)
		m.Private = m.Private || slices.Contains(list, privateClassifier)
	case slices.Equal(path, pathDependencies):
		err = m.entries(file, key, value, workspace.KindRuntime)
	case len(path) == len(pathOptional)+1 && slices.Equal(path[:len(pathOptional)], pathOptional):
		err = m.entries(file, key, value, workspace.KindOptional)
	case len(path) == 2 && path[0] == keyGroups, slices.Equal(path, pathDev):
		err = m.entries(file, key, value, workspace.KindDev)
	case slices.Equal(path, pathPackage):
		if value.Kind != unstable.Bool {
			return fmt.Errorf("%w: %s: %s, which is not a boolean", ErrManifest, file, key)
		}
		m.Private = m.Private || string(value.Data) == falseValue
	case slices.Equal(path, pathMembers):
		m.Members, err = texts(file, key, value)
	case slices.Equal(path, pathExclude):
		m.Exclude, err = texts(file, key, value)
	}
	return err
}

// mark records in m the tables that the key path is in: the table project and the table
// tool.uv.workspace.
func (m *Manifest) mark(path []string) {
	m.Project = m.Project || (len(path) > 0 && path[0] == keyProject)
	m.Workspace = m.Workspace ||
		(len(path) >= len(pathWorkspace) && slices.Equal(path[:len(pathWorkspace)], pathWorkspace))
}

// entries adds each requirement of the list value at key of file to the requirements of m, of the
// section kind. A dependency group may also include another group with an inline table, which adds
// no requirement. It returns an error that wraps [ErrManifest] for a value that is no list, for an
// element that is no string, and for a requirement that does not parse.
func (m *Manifest) entries(file, key string, value *unstable.Node, kind workspace.Kind) error {
	if value.Kind != unstable.Array {
		return fmt.Errorf("%w: %s: %s, which is not a list", ErrManifest, file, key)
	}
	it := value.Children()
	for it.Next() {
		n := it.Node()
		if n.Kind == unstable.InlineTable && kind == workspace.KindDev {
			continue
		}
		s, err := text(file, key, n)
		if err != nil {
			return err
		}
		r, err := ParseRequirement(s)
		if err != nil {
			return fmt.Errorf("%w: %s: %s: %w", ErrManifest, file, key, err)
		}
		m.Requirements = append(m.Requirements, Entry{Requirement: r, Text: s, Kind: kind, Span: span(n)})
	}
	return nil
}

// keys returns the parts of the key of it, in order.
func keys(it unstable.Iterator) []string {
	var out []string
	for it.Next() {
		out = append(out, string(it.Node().Data))
	}
	return out
}

// text returns the string value at key of file. It returns an error that wraps [ErrManifest] for a
// value that is no string.
func text(file, key string, value *unstable.Node) (string, error) {
	if value.Kind != unstable.String {
		return "", fmt.Errorf("%w: %s: %s, which is not a string", ErrManifest, file, key)
	}
	return string(value.Data), nil
}

// texts returns the strings of the list value at key of file. It returns an error that wraps
// [ErrManifest] for a value that is no list of strings.
func texts(file, key string, value *unstable.Node) ([]string, error) {
	if value.Kind != unstable.Array {
		return nil, fmt.Errorf("%w: %s: %s, which is not a list", ErrManifest, file, key)
	}
	var out []string
	it := value.Children()
	for it.Next() {
		s, err := text(file, key, it.Node())
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// span returns the range of the bytes of the value n in its file.
func span(n *unstable.Node) Span {
	return Span{Start: int(n.Raw.Offset), End: int(n.Raw.Offset + n.Raw.Length)}
}
