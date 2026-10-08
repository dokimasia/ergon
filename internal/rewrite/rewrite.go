// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package rewrite

import (
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/pin"
)

// The errors of [Apply].
var (
	// ErrNoOptions is the error for a package without the Options method of the producer.
	ErrNoOptions = errors.New("rewrite: the package has no Options method of the producer")

	// ErrNotLiteral is the error for a pin whose value is no literal, such as a value that a function
	// computes or a field that the literal leaves out.
	ErrNotLiteral = errors.New("rewrite: the value of the pin is no literal")

	// ErrMismatch is the error for a literal whose value differs from the value of its pin, and for a
	// release without the digest of a platform of the pin.
	ErrMismatch = errors.New("rewrite: the literal differs from the pin")
)

// optionsMethod is the method of a producer that returns its baseline.
const optionsMethod = "Options"

// The keys of the composite literals of an action and of a release binary that Apply rewrites.
const (
	commitKey  = "Commit"
	releaseKey = "Release"
	versionKey = "Version"
	digestsKey = "SHA256"
)

// binaryKey is the key of the embedded option.Binary in the literal of a release binary.
var binaryKey = reflect.TypeFor[option.Binary]().Name()

// Update is a pin and the release that it moves to.
type Update struct {
	// Release is the release.
	Release pin.Release

	// Pin is the pin.
	Pin pin.Pin
}

// source is a Go file of the package of a producer.
type source struct {
	// file is the syntax tree of data.
	file *ast.File

	// path is the path of the file.
	path string

	// data is the content of the file.
	data []byte
}

// change is a key of a composite literal, the value that the string literal of the key must have,
// and the value that replaces it.
type change struct {
	key, old, next string
}

// edit replaces the bytes from start to end of the file at path with text.
type edit struct {
	// path is the path of the file.
	path string

	// text is the replacement.
	text string

	// start and end are the offsets of the replaced bytes.
	start, end int
}

// pkg is the package of a producer.
type pkg struct {
	// fset has the positions of the files.
	fset *token.FileSet

	// vars are the values of the package-level variables, by name.
	vars map[string]ast.Expr

	// sources are the Go files of the package other than its test files, sorted by name.
	sources []source
}

// Apply writes updates into the Go files of the package in dir, other than its test files, and
// returns the paths of the files that it changed, sorted. In the Options method of the type
// producer, it follows the field names of each pin from the expression that the method returns
// through composite literals, & and parentheses, and through a package-level variable that a value
// names. It then replaces the string literals of the pin with the values of its release:
//
//   - a tool, as <package>@<version>, and a version with a source tag, with the version
//   - an action, its Commit and its Release, with the commit and the version
//   - a release binary, its Version, and each digest of its SHA256 with the digest that the release
//     states for the platforms whose earlier digest it is, in a literal with the key Binary or with
//     the keys of option.Binary itself
//
// The files keep their comments, and Apply formats each file that it changes with gofmt. It writes
// nothing when an update fails, and it writes the files one after another, so a write that fails
// leaves the earlier files written.
//
// It returns an error that wraps [ErrNoOptions] for a package without the Options method of
// producer, [ErrNotLiteral] for a pin whose value is no literal, [ErrMismatch] for a literal whose
// value differs from the pin, and the errors of reading, parsing and writing a file.
func Apply(dir, producer string, updates []Update) ([]string, error) {
	p, err := load(dir)
	if err != nil {
		return nil, err
	}
	root, ok := p.options(producer)
	if !ok {
		return nil, fmt.Errorf("%w: %s in %s", ErrNoOptions, producer, dir)
	}
	var edits []edit
	for i := range updates {
		e, err := p.rewrite(root, &updates[i])
		if err != nil {
			return nil, fmt.Errorf("rewrite: %s: %w", updates[i].Pin.Key, err)
		}
		edits = append(edits, e...)
	}
	return p.write(edits)
}

// load parses the Go files of the package in dir other than its test files, and collects the values
// of its package-level variables that a declaration states one by one. It returns the errors of
// reading the directory, and of reading and parsing a file.
func load(dir string) (*pkg, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("rewrite: read %s: %w", dir, err)
	}
	p := &pkg{fset: token.NewFileSet(), vars: map[string]ast.Expr{}}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("rewrite: read %s: %w", path, err)
		}
		f, err := parser.ParseFile(p.fset, path, data, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("rewrite: %w", err)
		}
		p.sources = append(p.sources, source{file: f, path: path, data: data})
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, s := range g.Specs {
				v, _ := s.(*ast.ValueSpec)
				if len(v.Names) != len(v.Values) {
					continue
				}
				for i, name := range v.Names {
					p.vars[name.Name] = v.Values[i]
				}
			}
		}
	}
	return p, nil
}

// options returns the expression that the Options method of the type producer returns, and reports
// whether the package has the method. The expression is the result of the return statement that
// ends the method, and nil for a method that ends in no return of one result.
func (p *pkg) options(producer string) (ast.Expr, bool) {
	for _, s := range p.sources {
		for _, d := range s.file.Decls {
			f, isFunc := d.(*ast.FuncDecl)
			if !isFunc || f.Name.Name != optionsMethod || f.Recv == nil {
				continue
			}
			recv := f.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if name, ok := recv.(*ast.Ident); !ok || name.Name != producer {
				continue
			}
			if f.Body == nil || len(f.Body.List) == 0 {
				return nil, true
			}
			r, ok := f.Body.List[len(f.Body.List)-1].(*ast.ReturnStmt)
			if !ok || len(r.Results) != 1 {
				return nil, true
			}
			return r.Results[0], true
		}
	}
	return nil, false
}

// rewrite returns the edits that move the pin of u, whose fields start at root, to the release of
// u. It returns an error that wraps ErrNotLiteral for a pin whose value is no literal, and
// ErrMismatch for a literal whose value differs from the pin.
func (p *pkg) rewrite(root ast.Expr, u *Update) ([]edit, error) {
	e := root
	for _, name := range u.Pin.Field {
		v, err := p.field(e, name)
		if err != nil {
			return nil, err
		}
		e = v
	}
	switch u.Pin.Kind {
	case pin.KindAction:
		a, _ := u.Pin.Value.(workflow.Action)
		return p.keys(e, []change{
			{key: commitKey, old: a.Commit, next: u.Release.Commit},
			{key: releaseKey, old: a.Release, next: u.Release.Version},
		})
	case pin.KindBinary:
		return p.binary(e, u)
	default:
		old := reflect.ValueOf(u.Pin.Value).String()
		ed, err := p.text(e, old, strings.TrimSuffix(old, u.Pin.Version)+u.Release.Version)
		if err != nil {
			return nil, err
		}
		return []edit{ed}, nil
	}
}

// binary returns the edits that move the release binary of u, whose literal e denotes, to the
// release of u: its version, and each digest. A digest takes the digest that the release states
// for the platforms whose earlier digest it is. It returns an error that wraps ErrNotLiteral for a
// value that is no literal, and ErrMismatch for a pin that is no release binary, for a literal whose
// value differs from the pin, and for a digest whose platforms the release gives no digest or two.
func (p *pkg) binary(e ast.Expr, u *Update) ([]edit, error) {
	r, ok := u.Pin.Value.(option.Release)
	if !ok {
		return nil, fmt.Errorf("%w: the pin is no release binary", ErrMismatch)
	}
	b := r.Pin()
	if embedded, err := p.field(e, binaryKey); err == nil {
		e = embedded
	}
	edits, err := p.keys(e, []change{{key: versionKey, old: b.Version, next: u.Release.Version}})
	if err != nil {
		return nil, err
	}
	digests, err := p.field(e, digestsKey)
	if err != nil {
		return nil, err
	}
	lit, ok := p.resolve(digests).(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("%w: the digests are no composite literal", ErrNotLiteral)
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("%w: a digest has no platform", ErrNotLiteral)
		}
		digest, old, err := p.literal(kv.Value)
		if err != nil {
			return nil, err
		}
		var next []string
		for platform, earlier := range b.SHA256 {
			if earlier == old {
				next = append(next, u.Release.Digests[platform])
			}
		}
		// Compact leaves two or more digests of a list unless every digest of it is the same.
		if next = slices.Compact(next); len(next) != 1 || next[0] == "" {
			return nil, fmt.Errorf("%w: the release has no one digest for the platforms of %s", ErrMismatch, old)
		}
		edits = append(edits, p.replace(digest, next[0]))
	}
	return edits, nil
}

// keys returns the edits of the string literals of the keys of changes in the composite literal
// that e denotes. It returns the errors that [pkg.field] and [pkg.text] return.
func (p *pkg) keys(e ast.Expr, changes []change) ([]edit, error) {
	edits := make([]edit, 0, len(changes))
	for _, c := range changes {
		v, err := p.field(e, c.key)
		if err != nil {
			return nil, err
		}
		ed, err := p.text(v, c.old, c.next)
		if err != nil {
			return nil, err
		}
		edits = append(edits, ed)
	}
	return edits, nil
}

// field returns the value of the key name in the composite literal that e denotes. It returns an
// error that wraps ErrNotLiteral for an e that denotes no composite literal, and for a literal
// without the key.
func (p *pkg) field(e ast.Expr, name string) (ast.Expr, error) {
	lit, ok := p.resolve(e).(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("%w: the field %s is in no composite literal", ErrNotLiteral, name)
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == name {
			return kv.Value, nil
		}
	}
	return nil, fmt.Errorf("%w: the literal has no field %s", ErrNotLiteral, name)
}

// text returns the edit that replaces the string literal that e denotes, whose value must be old,
// with next. It returns an error that wraps ErrNotLiteral for an e that denotes no string literal,
// and ErrMismatch for a literal with another value than old.
func (p *pkg) text(e ast.Expr, old, next string) (edit, error) {
	lit, value, err := p.literal(e)
	if err != nil {
		return edit{}, err
	}
	if value != old {
		return edit{}, fmt.Errorf("%w: the literal is %q, and the pin %q", ErrMismatch, value, old)
	}
	return p.replace(lit, next), nil
}

// literal returns the string literal that e denotes and its value. It returns an error that wraps
// ErrNotLiteral for an e that denotes no string literal.
func (p *pkg) literal(e ast.Expr) (*ast.BasicLit, string, error) {
	lit, ok := p.resolve(e).(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil, "", fmt.Errorf("%w: the value is no string literal", ErrNotLiteral)
	}
	// The parser accepts only a string literal that unquotes.
	value, _ := strconv.Unquote(lit.Value)
	return lit, value, nil
}

// replace returns the edit that replaces lit with the string literal of next.
func (p *pkg) replace(lit *ast.BasicLit, next string) edit {
	f := p.fset.File(lit.Pos())
	return edit{path: f.Name(), text: strconv.Quote(next), start: f.Offset(lit.Pos()), end: f.Offset(lit.End())}
}

// resolve returns the expression that e denotes: e without parentheses and &, and the value of the
// package-level variable that an identifier names. It returns nil for another operator than &, and
// for an identifier of no package-level variable.
func (p *pkg) resolve(e ast.Expr) ast.Expr {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.UnaryExpr:
			if x.Op != token.AND {
				return nil
			}
			e = x.X
		case *ast.Ident:
			e = p.vars[x.Name]
		default:
			return e
		}
	}
}

// write applies edits to the files of p, formats each file that they change, and writes it. It
// returns the paths of the files that it wrote, in the order of the files of p, and the error of a
// write.
func (p *pkg) write(edits []edit) ([]string, error) {
	var written []string
	for _, s := range p.sources {
		var own []edit
		for _, e := range edits {
			if e.path == s.path {
				own = append(own, e)
			}
		}
		if len(own) == 0 {
			continue
		}
		slices.SortFunc(own, func(a, b edit) int { return cmp.Compare(b.start, a.start) })
		data := s.data
		for _, e := range own {
			data = slices.Concat(data[:e.start], []byte(e.text), data[e.end:])
		}
		// Each edit replaces a string literal with a string literal, so the file still parses.
		formatted, _ := format.Source(data)
		if err := os.WriteFile(s.path, formatted, 0o600); err != nil {
			return written, fmt.Errorf("rewrite: write %s: %w", s.path, err)
		}
		written = append(written, s.path)
	}
	return written, nil
}
