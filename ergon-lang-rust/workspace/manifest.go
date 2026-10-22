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

// The keys of Cargo.toml that a release reads.
const (
	keyPackage      = "package"
	keyName         = "name"
	keyVersion      = "version"
	keyPublish      = "publish"
	keyWorkspace    = "workspace"
	keyMembers      = "members"
	keyExclude      = "exclude"
	keyDependencies = "dependencies"
	keyTarget       = "target"
	keyPath         = "path"
	keyOptional     = "optional"
)

// cratesIO is the name of crates.io in the registries of the key publish.
const cratesIO = "crates-io"

// keySeparator joins the parts of the key of a table in the index of the dependencies of a parser.
// A key of TOML can have any other character.
const keySeparator = "\x00"

// The paths of the keys of Cargo.toml that a release reads.
var (
	pathName             = []string{keyPackage, keyName}
	pathVersion          = []string{keyPackage, keyVersion}
	pathInheritedVersion = []string{keyPackage, keyVersion, keyWorkspace}
	pathPublish          = []string{keyPackage, keyPublish}
	pathMembers          = []string{keyWorkspace, keyMembers}
	pathExclude          = []string{keyWorkspace, keyExclude}
	pathWorkspaceVersion = []string{keyWorkspace, keyPackage, keyVersion}
	pathWorkspaceDeps    = []string{keyWorkspace, keyDependencies}
)

// sections are the tables of the dependencies of a crate, each with the kind of its requirements.
// Cargo also reads the spellings with an underscore of the editions before 2024.
var sections = map[string]workspace.Kind{
	keyDependencies:      workspace.KindRuntime,
	"build-dependencies": workspace.KindRuntime,
	"build_dependencies": workspace.KindRuntime,
	"dev-dependencies":   workspace.KindDev,
	"dev_dependencies":   workspace.KindDev,
}

// Span is the range of bytes of a value in a file, from Start to End.
type Span struct {
	Start, End int
}

// Dependency is a dependency of a crate, or of a workspace, as its Cargo.toml states it.
type Dependency struct {
	// Key is the key of the dependency in its table: the name of its crate, or the name under which
	// the code uses the crate of Package.
	Key string

	// Package is the key package, the name of the crate that Key renames, and empty without one.
	Package string

	// Path is the key path, the directory of the crate relative to the directory of the file, and
	// empty without one.
	Path string

	// Version is the version requirement: the key version, or the string of a dependency that the
	// file writes as a string. It is empty without one.
	Version string

	// Kind is the kind of the requirement: [workspace.KindRuntime] for dependencies and
	// build-dependencies, [workspace.KindOptional] for those with optional = true, and
	// [workspace.KindDev] for dev-dependencies. It is empty for a dependency of the workspace.
	Kind workspace.Kind

	// VersionSpan is the literal of Version in the file, with its quotes, and the zero Span without
	// a version.
	VersionSpan Span

	// Workspace reports the key workspace = true: the crate takes the path, the version and the
	// key package of the dependency from workspace.dependencies of the root of its workspace.
	Workspace bool
}

// Manifest is the content of a Cargo.toml that a release reads.
type Manifest struct {
	// Name is package.name as the file writes it.
	Name string

	// Version is package.version as the file writes it, and empty for a file without one and for an
	// inherited version.
	Version string

	// WorkspaceVersion is workspace.package.version as the file writes it, and empty without one.
	WorkspaceVersion string

	// Members and Exclude are the paths of workspace.members, which may be globs, and of
	// workspace.exclude.
	Members, Exclude []string

	// Dependencies are the dependencies of the crate in the order of the file: those of the tables
	// dependencies, dev-dependencies and build-dependencies, and those of the tables of each target.
	// A crate that a table names twice has two dependencies.
	Dependencies []Dependency

	// WorkspaceDependencies are the dependencies of workspace.dependencies in the order of the file,
	// which a crate of the workspace inherits with the key workspace.
	WorkspaceDependencies []Dependency

	// VersionSpan and WorkspaceVersionSpan are the literals of Version and of WorkspaceVersion in
	// the file, with their quotes.
	VersionSpan, WorkspaceVersionSpan Span

	// Package reports a table package, so the file is the manifest of a crate.
	Package bool

	// Workspace reports a table workspace, so the file is the root of a workspace.
	Workspace bool

	// InheritsVersion reports that package.version.workspace is true, so the crate has the version
	// workspace.package.version of the root of its workspace.
	InheritsVersion bool

	// Private reports a crate that crates.io does not receive: publish is false, or a list of
	// registries without crates-io.
	Private bool
}

// parser reads the values of the keys of a Cargo.toml into a manifest.
type parser struct {
	// m is the manifest that the parser fills.
	m *Manifest

	// entries are the indexes of the dependencies of m in Dependencies and in WorkspaceDependencies,
	// by the key of their table and their own key, joined with keySeparator.
	entries map[string]int

	// file is the name of the file in the errors.
	file string
}

// ParseManifest returns the manifest of the Cargo.toml data. file is the name of the file in the
// errors. ParseManifest reads the keys of its tables, of their dotted keys and of their inline
// tables, and skips each array of tables, such as the tables bin.
//
// It returns an error that wraps [ErrManifest] and names the file and the key for data that does
// not parse, and for a value of the wrong type.
func ParseManifest(file string, data []byte) (*Manifest, error) {
	p := parser{m: &Manifest{}, entries: map[string]int{}, file: file}
	var toml unstable.Parser
	toml.Reset(data)
	var table []string
	array := false
	for toml.NextExpression() {
		// The parser does not keep comments, so an expression is a key, a table or an array of tables.
		e := toml.Expression()
		if e.Kind != unstable.KeyValue {
			table, array = keys(e.Key()), e.Kind == unstable.ArrayTable
			p.mark(table)
			continue
		}
		if array {
			continue
		}
		if err := p.read(slices.Concat(table, keys(e.Key())), e.Value()); err != nil {
			return nil, err
		}
	}
	if err := toml.Error(); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrManifest, file, err)
	}
	return p.m, nil
}

// read reads the value at the key path into the manifest of p, and the value of each key of an
// inline table. It returns an error that wraps [ErrManifest] for a value of the wrong type.
func (p *parser) read(path []string, value *unstable.Node) error {
	p.mark(path)
	if value.Kind == unstable.InlineTable {
		it := value.Children()
		for it.Next() {
			kv := it.Node()
			if err := p.read(slices.Concat(path, keys(kv.Key())), kv.Value()); err != nil {
				return err
			}
		}
		return nil
	}
	m := p.m
	key := strings.Join(path, ".")
	var err error
	switch {
	case slices.Equal(path, pathName):
		m.Name, err = text(p.file, key, value)
	case slices.Equal(path, pathVersion):
		m.Version, err = text(p.file, key, value)
		m.VersionSpan = span(value)
	case slices.Equal(path, pathInheritedVersion):
		m.InheritsVersion, err = boolean(p.file, key, value)
	case slices.Equal(path, pathPublish):
		m.Private, err = private(p.file, key, value)
	case slices.Equal(path, pathMembers):
		m.Members, err = texts(p.file, key, value)
	case slices.Equal(path, pathExclude):
		m.Exclude, err = texts(p.file, key, value)
	case slices.Equal(path, pathWorkspaceVersion):
		m.WorkspaceVersion, err = text(p.file, key, value)
		m.WorkspaceVersionSpan = span(value)
	case len(path) > len(pathWorkspaceDeps) && slices.Equal(path[:len(pathWorkspaceDeps)], pathWorkspaceDeps):
		err = p.dependency(&m.WorkspaceDependencies, len(pathWorkspaceDeps), "", path, value)
	default:
		// The table of a dependency is a section, or a section of a target after the key target and
		// the configuration of the target.
		at := 0
		if len(path) > 2 && path[0] == keyTarget {
			at = 2
		}
		if kind, ok := sections[path[at]]; ok && len(path) > at+1 {
			err = p.dependency(&m.Dependencies, at+1, kind, path, value)
		}
	}
	return err
}

// mark records in the manifest of p the tables that the key path is in: the table package and the
// table workspace.
func (p *parser) mark(path []string) {
	p.m.Package = p.m.Package || (len(path) > 0 && path[0] == keyPackage)
	p.m.Workspace = p.m.Workspace || (len(path) > 0 && path[0] == keyWorkspace)
}

// dependency reads the value at the key path into a dependency of deps of the section kind. The
// table of the dependency is path[:at], and its key is path[at]. A value at the key itself is a
// version requirement as a string, and a value at a key after it is a key of the table of the
// dependency. dependency appends the dependency to deps when it reads the first key of it. It
// returns an error that wraps [ErrManifest] for a value of the wrong type.
func (p *parser) dependency(
	deps *[]Dependency,
	at int,
	kind workspace.Kind,
	path []string,
	value *unstable.Node,
) error {
	entry := strings.Join(path[:at+1], keySeparator)
	i, ok := p.entries[entry]
	if !ok {
		i = len(*deps)
		p.entries[entry] = i
		*deps = append(*deps, Dependency{Key: path[at], Kind: kind})
	}
	d := &(*deps)[i]
	key := strings.Join(path, ".")
	field := keyVersion
	if len(path) > at+1 {
		field = path[at+1]
	} else if value.Kind != unstable.String {
		return fmt.Errorf("%w: %s: %s, which is neither a version requirement nor a table", ErrManifest, p.file, key)
	}
	var err error
	switch field {
	case keyVersion:
		d.Version, err = text(p.file, key, value)
		d.VersionSpan = span(value)
	case keyPath:
		d.Path, err = text(p.file, key, value)
	case keyPackage:
		d.Package, err = text(p.file, key, value)
	case keyWorkspace:
		d.Workspace, err = boolean(p.file, key, value)
	case keyOptional:
		var optional bool
		optional, err = boolean(p.file, key, value)
		if optional && d.Kind == workspace.KindRuntime {
			d.Kind = workspace.KindOptional
		}
	}
	return err
}

// private reports whether the value of publish at key of file keeps a crate from crates.io: false,
// or a list of registries without crates-io. It returns an error that wraps [ErrManifest] for a
// value that is neither a boolean nor a list of strings.
func private(file, key string, value *unstable.Node) (bool, error) {
	if value.Kind == unstable.Bool {
		return string(value.Data) == "false", nil
	}
	registries, err := texts(file, key, value)
	return !slices.Contains(registries, cratesIO), err
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

// boolean returns the boolean value at key of file. It returns an error that wraps [ErrManifest]
// for a value that is no boolean.
func boolean(file, key string, value *unstable.Node) (bool, error) {
	if value.Kind != unstable.Bool {
		return false, fmt.Errorf("%w: %s: %s, which is not a boolean", ErrManifest, file, key)
	}
	return string(value.Data) == "true", nil
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
