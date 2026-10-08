// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"text/template"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
)

// ErrInvalidTemplate is the error for templates that a producer declares wrong: a template outside
// managed/, seeded/ and shared/, a name that does not end in .tmpl, a path that is not relative,
// clean and slash-separated or that is under .ergon, a template that does not parse, and a path
// that two producers render, unless each renders a fragment of a shared file. It is also the error
// for a file of a [language.Placer] at such a path, or at the path of another file. It is a defect
// of the producer.
var ErrInvalidTemplate = errors.New("render: invalid template")

// The delimiters of every template, so the ${{ }} of a workflow and the {{.Dir}} of go list -f in a
// Makefile are text.
const (
	left  = "{{%"
	right = "%}}"
)

// suffix ends the name of every template.
const suffix = ".tmpl"

// reserved is the directory of the lock and the local files, which no producer renders into.
const reserved = ".ergon/"

// backslash separates the elements of a path on Windows. A path of the repository is
// slash-separated, so it contains none.
const backslash = `\`

// Class is how ergon init treats a file that it renders.
type Class uint8

// The classes of a file.
const (
	// Managed is a file that ergon init renders whole and checks against its lock: the rendering of a
	// template under managed/, or the joined fragments of the templates under shared/.
	Managed Class = 1

	// Configured is a file whose keys ergon init writes, and whose every other key belongs to the
	// repository: .ergon.yaml.
	Configured Class = 2

	// Seeded is a file that ergon init writes when it is absent, and that the repository maintains
	// from then on: the rendering of a template under seeded/.
	Seeded Class = 3
)

// tree is a directory of a producer's templates.
type tree struct {
	// dir is the directory, with a slash.
	dir string

	// class is the class of the files of its templates.
	class Class
}

// shared is the directory of the templates of the fragments of shared files.
const shared = "shared/"

// trees are the directories of a producer's templates.
var trees = []tree{{dir: "managed/", class: Managed}, {dir: "seeded/", class: Seeded}, {dir: shared, class: Managed}}

// Unit is a producer of a rendering: its name, the producer, and its resolved options.
type Unit struct {
	// Producer renders the templates of the unit.
	Producer language.Producer

	// Options are the resolved options of the producer, or nil for a producer that is not
	// [language.Configurable].
	Options language.Options

	// Name is the name of the producer in the lock, such as common or go.
	Name string
}

// File is a file of a rendering: the rendering of a template, or the joined fragments of a shared
// file.
type File struct {
	// Path is the path of the file in the repository.
	Path string

	// Producer is the name of the producer of the file, or of its first fragment.
	Producer string

	// Content is the rendering.
	Content []byte

	// Class is how ergon init treats the file.
	Class Class

	// Shared reports that the file joins the fragments of several producers.
	Shared bool
}

// data is what a template reads.
type data struct {
	// Answers are the answers of ergon init.
	Answers *language.Answers

	// Options are the options of the template's producer, or nil.
	Options language.Options

	// Data are the values that the template's producer computes, or nil.
	Data any

	// Contributions are the contributions of every producer to the workflows.
	Contributions workflow.Contribution
}

// Render returns the files that units render for a and the contributions c, sorted by path. It
// executes the templates of each unit with the delimiters {{% and %}}, the functions words, make,
// yaml and steps, and the data .Answers, .Options, .Data and .Contributions, where a key that the
// data lacks is an error. .Data are the values that a [language.Calculator] computes from a, its
// options and c, and nil for any other producer. Render skips a template that renders no byte, and
// joins the fragments of a shared file in the order of units. It adds the files of a
// [language.Placer] as managed files of its unit, after the templates of the unit.
//
// It returns an error that wraps [ErrInvalidTemplate] for templates or placed files that a
// producer declares wrong, the error of the Data or the Files of a producer, and the error of a
// template that does not execute, which wraps the error of a function that it calls. Render reads a
// and c, and modifies neither.
func Render(units []Unit, a *language.Answers, c *workflow.Contribution) ([]File, error) {
	var files []File
	for _, u := range units {
		d := data{Answers: a, Options: u.Options, Contributions: *c}
		if calculator, ok := u.Producer.(language.Calculator); ok {
			values, err := calculator.Data(a, u.Options, c)
			if err != nil {
				return nil, fmt.Errorf("render: the data of %s: %w", u.Name, err)
			}
			d.Data = values
		}
		err := fs.WalkDir(u.Producer.Templates(), ".", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			f, err := execute(u.Producer.Templates(), name, &d)
			if err != nil || len(f.Content) == 0 {
				return err
			}
			f.Producer = u.Name
			i := slices.IndexFunc(files, func(g File) bool { return g.Path == f.Path })
			if i < 0 {
				files = append(files, f)
				return nil
			}
			if !files[i].Shared || !f.Shared {
				return fmt.Errorf("%w: %s and %s both render %s", ErrInvalidTemplate, files[i].Producer, u.Name, f.Path)
			}
			files[i].Content = append(files[i].Content, f.Content...)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("render: the templates of %s: %w", u.Name, err)
		}
		if files, err = place(files, &u, a, c); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return files, nil
}

// place returns files with the files that the producer of u places for a and c, as managed files of
// u, and files alone for a producer that is no [language.Placer]. It returns the error of Files,
// and an error that wraps [ErrInvalidTemplate] for a path that is not relative, clean and
// slash-separated, a path under .ergon, and a path that another file of files has.
func place(files []File, u *Unit, a *language.Answers, c *workflow.Contribution) ([]File, error) {
	placer, ok := u.Producer.(language.Placer)
	if !ok {
		return files, nil
	}
	placed, err := placer.Files(a, u.Options, c)
	if err != nil {
		return nil, fmt.Errorf("render: the files of %s: %w", u.Name, err)
	}
	for _, p := range placed {
		clean := fs.ValidPath(p.Path) && p.Path != "." && !strings.Contains(p.Path, backslash)
		if !clean || strings.HasPrefix(p.Path+"/", reserved) {
			return nil, fmt.Errorf("%w: %s places %q, which is no clean relative path outside %s",
				ErrInvalidTemplate, u.Name, p.Path, reserved)
		}
		if slices.ContainsFunc(files, func(f File) bool { return f.Path == p.Path }) {
			return nil, fmt.Errorf("%w: %s places %s, which another file of the rendering has", ErrInvalidTemplate,
				u.Name, p.Path)
		}
		files = append(files, File{Path: p.Path, Producer: u.Name, Content: p.Content, Class: Managed})
	}
	return files, nil
}

// execute returns the file of the template name of fsys for d: its path, its class, whether it is
// a fragment of a shared file, and its rendering. It returns an error that wraps
// [ErrInvalidTemplate] for a template outside managed/, seeded/ and shared/, a name without .tmpl,
// a path under .ergon, and a template that does not parse, and the error of a template that does not
// execute.
func execute(fsys fs.FS, name string, d *data) (File, error) {
	i := slices.IndexFunc(trees, func(t tree) bool { return strings.HasPrefix(name, t.dir) })
	path, ok := strings.CutSuffix(name, suffix)
	if i < 0 || !ok {
		return File{}, fmt.Errorf("%w: %s, which is not managed/, seeded/ or shared/<path>%s", ErrInvalidTemplate, name,
			suffix)
	}
	path = strings.TrimPrefix(path, trees[i].dir)
	if path == "" || strings.HasPrefix(path+"/", reserved) {
		return File{}, fmt.Errorf("%w: %s, whose path is empty or under %s", ErrInvalidTemplate, name, reserved)
	}
	text, err := fs.ReadFile(fsys, name)
	if err != nil {
		return File{}, fmt.Errorf("render: read %s: %w", name, err)
	}
	tmpl, err := template.New(name).Delims(left, right).Funcs(functions).Option("missingkey=error").Parse(string(text))
	if err != nil {
		return File{}, fmt.Errorf("%w: %w", ErrInvalidTemplate, err)
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, d); err != nil {
		return File{}, fmt.Errorf("render: %w", err)
	}
	return File{Path: path, Content: b.Bytes(), Class: trees[i].class, Shared: trees[i].dir == shared}, nil
}
