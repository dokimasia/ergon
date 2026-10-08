// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// errData is the error of the data of a producer of the cases.
var errData = errors.New("data: the options do not render")

// errFault is the error of an Open that a failing file system fails.
var errFault = errors.New("fault: injected")

// errPlace is the error of the files of a placer of the cases.
var errPlace = errors.New("place: the options name no directory")

// placed is the content of each file that a placer of the cases places.
const placed = "placed\n"

// fixture is a producer of the cases: its templates, and the data or the error that Data returns.
type fixture struct {
	// templates are the templates of the producer.
	templates fs.FS

	// data are the values that Data returns.
	data any

	// err is the error that Data returns.
	err error
}

// Templates returns f.templates.
func (f fixture) Templates() fs.FS {
	return f.templates
}

// Data returns f.data and f.err, whatever the answers, the options and the contributions.
func (f fixture) Data(*language.Answers, language.Options, *workflow.Contribution) (any, error) {
	return f.data, f.err
}

// identifiers is a producer of the cases whose data are the identifiers of the jobs of the
// contributions.
type identifiers struct {
	// templates are the templates of the producer.
	templates fs.FS
}

// Templates returns i.templates.
func (i identifiers) Templates() fs.FS {
	return i.templates
}

// Data returns the identifiers of the jobs of c, in their order.
func (identifiers) Data(_ *language.Answers, _ language.Options, c *workflow.Contribution) (any, error) {
	ids := make([]string, 0, len(c.Jobs))
	for _, j := range c.Jobs {
		ids = append(ids, j.ID)
	}
	return ids, nil
}

// plain is a producer of the cases that computes no values.
type plain struct {
	// templates are the templates of the producer.
	templates fs.FS
}

// Templates returns p.templates.
func (p plain) Templates() fs.FS {
	return p.templates
}

// placer is a producer of the cases that places files: its templates, and the files or the error
// that Files returns.
type placer struct {
	// templates are the templates of the producer.
	templates fs.FS

	// err is the error that Files returns.
	err error

	// files are the files that Files returns.
	files []language.File
}

// Templates returns p.templates.
func (p placer) Templates() fs.FS {
	return p.templates
}

// Files returns p.files and p.err, whatever the answers, the options and the contributions.
func (p placer) Files(*language.Answers, language.Options, *workflow.Contribution) ([]language.File, error) {
	return p.files, p.err
}

// options are the options of the producers of the cases.
type options struct {
	// Paths are the paths of the targets.
	Paths []string `yaml:"paths"`
}

// Validate returns nil.
func (*options) Validate() error {
	return nil
}

// failing is a file system whose Open fails for name. It hides the ReadFile and ReadDir of the
// file system that it wraps.
type failing struct {
	fs.FS

	// name is the path whose Open fails.
	name string
}

// Open fails for f.name, and opens any other file, with its error wrapped.
func (f failing) Open(name string) (fs.File, error) {
	if name == f.name {
		return nil, errFault
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failing: %w", err)
	}
	return file, nil
}

func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		t.Run("renders the class of each file from the directory of its template", func(t *testing.T) {
			t.Parallel()
			files, err := render.Render([]render.Unit{unit("common", fstest.MapFS{
				"managed/LICENSE.tmpl":                  {Data: []byte("Copyright {{% .Answers.Owner %}}\n")},
				"seeded/README.md.tmpl":                 {Data: []byte("# {{% .Answers.Name %}}\n")},
				"shared/.gitignore.tmpl":                {Data: []byte("bin/\n")},
				"managed/.github/workflows/ci.yml.tmpl": {Data: []byte("name: CI\n")},
			})}, answers(), &workflow.Contribution{})
			assert.NoError(t, err, "Render")
			assert.Equal(t, files, []render.File{
				{
					Path:     ".github/workflows/ci.yml",
					Producer: "common",
					Content:  []byte("name: CI\n"),
					Class:    render.Managed,
				},
				{
					Path:     ".gitignore",
					Producer: "common",
					Content:  []byte("bin/\n"),
					Class:    render.Managed,
					Shared:   true,
				},
				{
					Path:     "LICENSE",
					Producer: "common",
					Content:  []byte("Copyright Dokimasia B.V.\n"),
					Class:    render.Managed,
				},
				{Path: "README.md", Producer: "common", Content: []byte("# demo\n"), Class: render.Seeded},
			}, "the files")
		})

		t.Run("joins the fragments of a shared file in the order of the units", func(t *testing.T) {
			t.Parallel()
			files, err := render.Render([]render.Unit{
				unit("common", fstest.MapFS{"shared/.gitignore.tmpl": {Data: []byte("# common\n")}}),
				unit("go", fstest.MapFS{"shared/.gitignore.tmpl": {Data: []byte("bin/\n")}}),
				unit("python", fstest.MapFS{"shared/.gitignore.tmpl": {Data: []byte(".venv/\n")}}),
			}, answers(), &workflow.Contribution{})
			assert.NoError(t, err, "Render")
			assert.Equal(t, files, []render.File{
				{
					Path:     ".gitignore",
					Producer: "common",
					Content:  []byte("# common\nbin/\n.venv/\n"),
					Class:    render.Managed,
					Shared:   true,
				},
			}, "the files")
		})

		t.Run("skips a template that renders no byte", func(t *testing.T) {
			t.Parallel()
			files, err := render.Render([]render.Unit{unit("license", fstest.MapFS{
				"managed/NOTICE.tmpl": {Data: []byte(`{{% if eq .Answers.License "Apache-2.0" %}}notice{{% end %}}`)},
			})}, answers(), &workflow.Contribution{})
			assert.NoError(t, err, "Render")
			assert.Empty(t, files, "the files")
		})

		t.Run("renders the options, the data and the contributions", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{
				Name:    "go",
				Options: &options{Paths: []string{"./..."}},
				Producer: fixture{data: map[string]string{"pin": "go.work"}, templates: fstest.MapFS{
					"managed/x.tmpl": {Data: []byte("{{% words .Options.Paths %}} {{% .Data.pin %}} " +
						"{{% range .Contributions.Jobs %}}{{% .ID %}}{{% end %}}\n")},
				}},
			}
			c := workflow.Contribution{Jobs: []workflow.Job{{ID: "check-go"}}}
			files, err := render.Render([]render.Unit{u}, answers(), &c)
			assert.NoError(t, err, "Render")
			assert.Length(t, files, 1, "the files")
			assert.Equal(t, string(files[0].Content), "./... go.work check-go\n", "the rendering")
		})

		t.Run("computes the data of a producer from the contributions", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "github", Producer: identifiers{templates: fstest.MapFS{
				"managed/x.tmpl": {Data: []byte("{{% words .Data %}}\n")},
			}}}
			c := workflow.Contribution{Jobs: []workflow.Job{{ID: "docs"}, {ID: "check-go"}}}
			files, err := render.Render([]render.Unit{u}, answers(), &c)
			assert.NoError(t, err, "Render")
			assert.Length(t, files, 1, "the files")
			assert.Equal(t, string(files[0].Content), "docs check-go\n", "the rendering")
		})

		t.Run("leaves the expressions of a workflow and of go list as text", func(t *testing.T) {
			t.Parallel()
			text := "group: ${{ github.workflow }}\nlist: go list -m -f '{{.Dir}}'\n"
			files, err := render.Render(
				[]render.Unit{unit("github", fstest.MapFS{"managed/x.yml.tmpl": {Data: []byte(text)}})},
				answers(),
				&workflow.Contribution{},
			)
			assert.NoError(t, err, "Render")
			assert.Length(t, files, 1, "the files")
			assert.Equal(t, string(files[0].Content), text, "the rendering")
		})

		t.Run("renders no data for a producer that computes no values", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "go", Producer: plain{templates: fstest.MapFS{
				"managed/x.tmpl": {Data: []byte("{{% if .Data %}}data{{% else %}}none{{% end %}}\n")},
			}}}
			files, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
			assert.NoError(t, err, "Render")
			assert.Length(t, files, 1, "the files")
			assert.Equal(t, string(files[0].Content), "none\n", "the rendering")
		})

		t.Run("adds the files of a placer as managed files of its producer", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "license", Producer: placer{
				templates: fstest.MapFS{"managed/LICENSE.tmpl": {Data: []byte("MIT\n")}},
				files: []language.File{
					{Path: "enterprise/LICENSE", Content: []byte(placed)},
					{Path: "sdk/go/NOTICE", Content: []byte(placed)},
				},
			}}
			files, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
			assert.NoError(t, err, "Render")
			assert.Equal(t, files, []render.File{
				{Path: "LICENSE", Producer: "license", Content: []byte("MIT\n"), Class: render.Managed},
				{Path: "enterprise/LICENSE", Producer: "license", Content: []byte(placed), Class: render.Managed},
				{Path: "sdk/go/NOTICE", Producer: "license", Content: []byte(placed), Class: render.Managed},
			}, "the files")
		})

		t.Run("returns the error of the data of a producer", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "go", Producer: fixture{err: errData, templates: fstest.MapFS{}}}
			_, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
			assert.ErrorIs(t, err, errData, "Render")
		})

		t.Run("returns the error of the files of a placer", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "license", Producer: placer{err: errPlace, templates: fstest.MapFS{}}}
			_, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
			assert.ErrorIs(t, err, errPlace, "Render")
		})

		t.Run("returns the error of a template that does not execute", func(t *testing.T) {
			t.Parallel()
			u := render.Unit{Name: "go", Producer: fixture{data: map[string]string{}, templates: fstest.MapFS{
				"managed/x.tmpl": {Data: []byte("{{% .Data.missing %}}\n")},
			}}}
			_, err := render.Render([]render.Unit{u}, answers(), &workflow.Contribution{})
			assert.HasError(t, err, "Render")
			assert.Contains(t, err.Error(), "missing", "the error")
		})

		t.Run("returns an error for a template that does not read", func(t *testing.T) {
			t.Parallel()
			templates := failing{FS: fstest.MapFS{"managed/x.tmpl": {Data: []byte("x\n")}}, name: "managed/x.tmpl"}
			_, err := render.Render([]render.Unit{unit("go", templates)}, answers(), &workflow.Contribution{})
			assert.ErrorIs(t, err, errFault, "Render")
		})

		t.Run("returns an error for a directory of templates that does not read", func(t *testing.T) {
			t.Parallel()
			templates := failing{FS: fstest.MapFS{"managed/x.tmpl": {Data: []byte("x\n")}}, name: "managed"}
			_, err := render.Render([]render.Unit{unit("go", templates)}, answers(), &workflow.Contribution{})
			assert.ErrorIs(t, err, errFault, "Render")
		})

		invalid := []struct {
			name  string
			units []render.Unit
		}{
			{
				name:  "returns ErrInvalidTemplate for a template outside the three directories",
				units: []render.Unit{unit("go", fstest.MapFS{"Makefile.tmpl": {Data: []byte("x\n")}})},
			},
			{
				name:  "returns ErrInvalidTemplate for a name without the suffix .tmpl",
				units: []render.Unit{unit("go", fstest.MapFS{"managed/Makefile": {Data: []byte("x\n")}})},
			},
			{
				name:  "returns ErrInvalidTemplate for a template without a path",
				units: []render.Unit{unit("go", fstest.MapFS{"managed/.tmpl": {Data: []byte("x\n")}})},
			},
			{
				name:  "returns ErrInvalidTemplate for a path under .ergon",
				units: []render.Unit{unit("go", fstest.MapFS{"managed/.ergon/init.lock.tmpl": {Data: []byte("x\n")}})},
			},
			{
				name:  "returns ErrInvalidTemplate for a template that does not parse",
				units: []render.Unit{unit("go", fstest.MapFS{"managed/x.tmpl": {Data: []byte("{{% if %}}\n")}})},
			},
			{
				name: "returns ErrInvalidTemplate for a managed file that two producers render",
				units: []render.Unit{
					unit("common", fstest.MapFS{"managed/LICENSE.tmpl": {Data: []byte("a\n")}}),
					unit("license", fstest.MapFS{"managed/LICENSE.tmpl": {Data: []byte("b\n")}}),
				},
			},
			{
				name: "returns ErrInvalidTemplate for a fragment of a managed file",
				units: []render.Unit{
					unit("common", fstest.MapFS{"managed/Makefile.tmpl": {Data: []byte("a\n")}}),
					unit("go", fstest.MapFS{"shared/Makefile.tmpl": {Data: []byte("b\n")}}),
				},
			},
			{
				name: "returns ErrInvalidTemplate for a managed file of a shared path",
				units: []render.Unit{
					unit("common", fstest.MapFS{"shared/Makefile.tmpl": {Data: []byte("a\n")}}),
					unit("go", fstest.MapFS{"managed/Makefile.tmpl": {Data: []byte("b\n")}}),
				},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file without a path",
				units: []render.Unit{placing("license", "")},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file at an absolute path",
				units: []render.Unit{placing("license", "/LICENSE")},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file at a path that is not clean",
				units: []render.Unit{placing("license", "a/../LICENSE")},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file at the root directory",
				units: []render.Unit{placing("license", ".")},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file at a path with a backslash",
				units: []render.Unit{placing("license", `a\LICENSE`)},
			},
			{
				name:  "returns ErrInvalidTemplate for a placed file under .ergon",
				units: []render.Unit{placing("license", ".ergon/LICENSE")},
			},
			{
				name:  "returns ErrInvalidTemplate for two placed files at one path",
				units: []render.Unit{placing("license", "a/LICENSE", "a/LICENSE")},
			},
			{
				name: "returns ErrInvalidTemplate for a placed file at the path of a template of an earlier unit",
				units: []render.Unit{
					unit("common", fstest.MapFS{"managed/a/LICENSE.tmpl": {Data: []byte("a\n")}}),
					placing("license", "a/LICENSE"),
				},
			},
			{
				name: "returns ErrInvalidTemplate for a template at the path of a placed file of an earlier unit",
				units: []render.Unit{
					placing("license", "a/LICENSE"),
					unit("common", fstest.MapFS{"managed/a/LICENSE.tmpl": {Data: []byte("a\n")}}),
				},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := render.Render(tt.units, answers(), &workflow.Contribution{})
				assert.ErrorIs(t, err, render.ErrInvalidTemplate, "Render")
			})
		}
	})
}

// unit returns a unit of the cases without options and data, whose templates are templates.
func unit(name string, templates fs.FS) render.Unit {
	return render.Unit{Name: name, Producer: fixture{templates: templates}}
}

// placing returns a unit of the cases without templates, whose producer places a file of the
// content placed at each of paths.
func placing(name string, paths ...string) render.Unit {
	files := make([]language.File, 0, len(paths))
	for _, p := range paths {
		files = append(files, language.File{Path: p, Content: []byte(placed)})
	}
	return render.Unit{Name: name, Producer: placer{templates: fstest.MapFS{}, files: files}}
}

// answers returns the answers of the cases.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Owner:           "Dokimasia B.V.",
		License:         spdx.MIT,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
		Languages:       []workspace.Language{"go"},
		Year:            2026,
	}
}
