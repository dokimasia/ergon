// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// The toolchains and the languages of the cases. alpha and beta render files, and gamma registers
// no producer. The toolchain shared renders a file that its languages delta and epsilon share.
const (
	tool    workspace.Toolchain = "tool"
	alpha   workspace.Language  = "alpha"
	beta    workspace.Language  = "beta"
	gamma   workspace.Language  = "gamma"
	shared  workspace.Toolchain = "shared"
	delta   workspace.Language  = "delta"
	epsilon workspace.Language  = "epsilon"
)

// ownerSection is the name of the producer owner and of its section of .ergon.yaml.
const ownerSection = "owner"

// edited is the section of the producer owner whose key owner states another value than the
// answers of the cases, and editedKey is that key and value as the error of the options names them.
const (
	edited    = ownerSection + ":\n  owner: Someone Else\n"
	editedKey = ownerSection + `.owner "Someone Else"`
)

// The paths that the producers of the cases render, and the path of the lock.
const (
	ignore   = ".gitignore"
	license  = "LICENSE"
	readme   = "README.md"
	config   = ".ergon.yaml"
	ci       = ".github/ci.yml"
	greeting = "greeting.txt"
	alphaCfg = "alpha/settings.txt"
	betaCfg  = "beta.txt"
	lockPath = ".ergon/init.lock"

	// sharedFile is the file of the toolchain shared.
	sharedFile = "shared.txt"
)

// version is the version of ergon that the locks of the cases record.
const version = "1.2.3"

// workflowContent is the managed YAML file of common: a comment, a scalar, a map and a list.
const workflowContent = "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n"

// configured is the .ergon.yaml that New writes for the answers of the cases.
const configured = `common:
  # The greeting of greeting.txt.
  greeting: hello
alpha:
  # The greeting of greeting.txt.
  greeting: hi
`

// errRender is the error of a producer that cannot render its answers.
var errRender = errors.New("render: the answers do not render")

// producer is a producer of the cases: its templates, and the error of its data.
type producer struct {
	// templates are the templates of the producer.
	templates fstest.MapFS

	// err is the error that Data returns.
	err error
}

// Templates returns p.templates.
func (p producer) Templates() fs.FS {
	return p.templates
}

// Data returns no data and p.err.
func (p producer) Data(*language.Answers, language.Options, *workflow.Contribution) (any, error) {
	return nil, p.err
}

// greetings are the options of the configurable producers of the cases.
type greetings struct {
	// Greeting is what greeting.txt states.
	Greeting string `yaml:"greeting" doc:"The greeting of greeting.txt."`
}

// Validate returns nil.
func (*greetings) Validate() error {
	return nil
}

// configurable is a producer of the cases with options, whose baseline greeting states.
type configurable struct {
	producer

	// greeting is the greeting at the baseline.
	greeting string
}

// Options returns the options at the baseline.
func (c configurable) Options() language.Options {
	return &greetings{Greeting: c.greeting}
}

// contributing is a producer of the cases with a part of the workflows.
type contributing struct {
	producer

	// part is the contribution that Contribution returns.
	part workflow.Contribution
}

// Contribution returns c.part, whatever the options.
func (c contributing) Contribution(language.Options) workflow.Contribution {
	return c.part
}

// refusal is a value whose encoding in YAML returns errRender.
type refusal struct{}

// MarshalYAML returns errRender.
func (refusal) MarshalYAML() (any, error) {
	return nil, errRender
}

// refusing is options with a field whose encoding fails.
type refusing struct {
	// Value refuses its encoding.
	Value refusal `yaml:"value" doc:"A value that does not encode."`
}

// Validate returns nil.
func (*refusing) Validate() error {
	return nil
}

// unwritable is a producer whose options do not encode.
type unwritable struct {
	producer
}

// Options returns options that do not encode.
func (unwritable) Options() language.Options {
	return &refusing{}
}

// ownership is the options of the producer owner, whose key owner states the answer owner.
type ownership struct {
	// Owner is the answer owner.
	Owner string `yaml:"owner" doc:"The owner, an answer of ergon init." answer:"owner"`
}

// Validate returns nil.
func (*ownership) Validate() error {
	return nil
}

// owner is a base producer of the cases without templates, whose options state an answer.
type owner struct {
	producer
}

// Options returns the options at the baseline, without the answer.
func (owner) Options() language.Options {
	return &ownership{}
}

func TestRepository(t *testing.T) {
	t.Parallel()

	t.Run("Open", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			fsys      baseline.FS
			catalog   *language.Catalog
			version   string
			producers []baseline.Producer
		}{
			{name: "returns ErrInvalidOpen for a nil file system", catalog: catalog(t), version: version},
			{name: "returns ErrInvalidOpen for a nil catalog", fsys: directory(t), version: version},
			{name: "returns ErrInvalidOpen for an empty version", fsys: directory(t), catalog: catalog(t)},
			{
				name: "returns ErrInvalidOpen for a producer without a producer", fsys: directory(t),
				catalog: catalog(t), version: version, producers: []baseline.Producer{{Name: "common"}},
			},
			{
				name: "returns ErrInvalidOpen for a producer with an invalid name", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: "Common", Producer: common("hello").Producer}},
			},
			{
				name: "returns ErrInvalidOpen for a producer named as a language", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: string(alpha), Producer: common("hello").Producer}},
			},
			{
				name: "returns ErrInvalidOpen for a producer named as a toolchain", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: string(tool), Producer: common("hello").Producer}},
			},
			{
				name: "returns ErrInvalidOpen for two producers with one name", fsys: directory(t),
				catalog: catalog(t), version: version, producers: []baseline.Producer{common("hello"), common("hello")},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := baseline.Open(tt.fsys, tt.catalog, tt.version, tt.producers...)
				assert.ErrorIs(t, err, baseline.ErrInvalidOpen, "Open")
			})
		}
	})

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the managed and seeded files, the sections of .ergon.yaml and the lock", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			changes, err := repository(t, root).New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(t, paths(changes), []string{
				config, ci, ignore, license, readme, alphaCfg, greeting, lockPath,
			}, "the files that New wrote")
			expect := []struct{ name, want string }{
				{name: readme, want: "# demo\n"},
				{name: config, want: configured},
				{name: alphaCfg, want: "alpha\n"},
				{name: greeting, want: "hello\n"},
			}
			for _, e := range expect {
				assert.Equal(t, content(t, root, e.name), e.want, "the file "+e.name)
			}
		})

		t.Run("writes each field of the lock", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			want := `{
  "ergon": "1.2.3",
  "files": [
    {
      "path": ".github/ci.yml",
      "producer": "common",
      "sha256": "` + sum(workflowContent) + `"
    },
    {
      "path": ".gitignore",
      "producer": "common",
      "sha256": "` + sum("# common\nalpha/\n") + `"
    },
    {
      "path": "LICENSE",
      "producer": "common",
      "sha256": "` + sum("Copyright Dokimasia B.V.\n") + `"
    },
    {
      "path": "alpha/settings.txt",
      "producer": "alpha",
      "sha256": "` + sum("alpha\n") + `"
    },
    {
      "path": "greeting.txt",
      "producer": "common",
      "sha256": "` + sum("hello\n") + `"
    }
  ],
  "settings": {
    "alpha.greeting": "hi",
    "common.greeting": "hello"
  },
  "answers": {
    "name": "demo",
    "owner": "Dokimasia B.V.",
    "license": "MIT",
    "repository": "dokimasia/demo",
    "security-contact": "security@example.com",
    "languages": [
      "alpha"
    ],
    "year": 2026
  }
}
`
			assert.Equal(t, content(t, root, lockPath), want, "the lock")
		})

		t.Run("writes an empty list of files for a rendering without managed files", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			r, err := baseline.Open(root, catalog(t), version)
			assert.NoError(t, err, "Open")
			a := answers()
			a.Languages = nil
			_, err = r.New(a, baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Contains(t, content(t, root, lockPath), `"files": []`, "the lock")
			assert.False(t, exists(t, root, config), "the .ergon.yaml of a rendering without options")
		})

		t.Run("joins the fragments of a shared file in the order of the producers", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			a := answers()
			a.Languages = []workspace.Language{beta, alpha}
			_, err := repository(t, root).New(a, baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(t, content(t, root, ignore), "# common\nalpha/\nbeta/\n", "the joined .gitignore")
		})

		t.Run("renders the files of a shared toolchain once before its first language", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			r, err := baseline.Open(root, sharedCatalog(t), version, common("hello"))
			assert.NoError(t, err, "Open")
			a := answers()
			a.Languages = []workspace.Language{epsilon, delta}
			_, err = r.New(a, baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(t, content(t, root, ignore), "# common\nshared/\ndelta/\nepsilon/\n", "the joined .gitignore")
			assert.Equal(t, content(t, root, sharedFile), "shared\n", "the file of the toolchain")
			assert.Contains(t, content(t, root, lockPath), `"path": "shared.txt",
      "producer": "shared"`, "the lock")
		})

		t.Run("keeps the keys of an existing .ergon.yaml that are no section", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, config, "# ours\nchecks:\n  coverage: 100\n")
			_, err := repository(t, root).New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(
				t,
				content(t, root, config),
				"# ours\nchecks:\n  coverage: 100\n"+configured,
				"the .ergon.yaml",
			)
		})

		t.Run("renders the value of an option of an existing .ergon.yaml", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, config, "common:\n  greeting: hey\n")
			_, err := repository(t, root).New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(t, content(t, root, greeting), "hey\n", "the greeting")
		})

		t.Run("writes the answer into a key of an existing .ergon.yaml that states another value", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, config, edited)
			r, err := baseline.Open(root, catalog(t), version, common("hello"),
				baseline.Producer{Name: ownerSection, Producer: owner{}})
			assert.NoError(t, err, "Open")
			_, err = r.New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Contains(t, content(t, root, config), "owner: Dokimasia B.V.\n", "the .ergon.yaml")
		})

		t.Run("returns ErrInitialized for a repository with a lock", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrInitialized, "a second New")
		})

		t.Run("returns ErrInvalid of the lock for a lock that does not parse", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, lockPath, "not JSON\n")
			_, err := r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, lock.ErrInvalid, "New over a lock that does not parse")
		})

		t.Run("returns ErrConflict for a managed file with other content", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, license, "Copyright someone else\n")
			_, err := repository(t, root).New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrConflict, "New")
			assert.Contains(t, err.Error(), license, "the paths of the conflict")
			assert.False(t, exists(t, root, lockPath), "the lock after the conflict")
		})

		t.Run("overwrites a managed file with other content with Force", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, license, "Copyright someone else\n")
			_, err := repository(t, root).New(answers(), baseline.Options{Force: true})
			assert.NoError(t, err, "New with Force")
			assert.Equal(t, content(t, root, license), "Copyright Dokimasia B.V.\n", "the LICENSE")
		})

		t.Run("keeps a seeded file that exists", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, readme, "# Our own README\n")
			changes, err := repository(t, root).New(answers(), baseline.Options{Force: true})
			assert.NoError(t, err, "New")
			assert.NotContains(t, paths(changes), readme, "the files that New wrote")
			assert.Equal(t, content(t, root, readme), "# Our own README\n", "the README.md")
		})

		t.Run("leaves a managed file that equals the rendering", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, license, "Copyright Dokimasia B.V.\n")
			changes, err := repository(t, root).New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.NotContains(t, paths(changes), license, "the files that New wrote")
		})

		t.Run("returns ErrInvalidAnswer for answers that no producer can render", func(t *testing.T) {
			t.Parallel()
			a := answers()
			a.License = "AMD-newlib"
			_, err := repository(t, directory(t)).New(a, baseline.Options{})
			assert.ErrorIs(t, err, language.ErrInvalidAnswer, "New")
		})

		t.Run("returns ErrInvalid of the options for a key that a section does not have", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			put(t, root, config, "common:\n  greting: hey\n")
			_, err := repository(t, root).New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, options.ErrInvalid, "New")
		})

		t.Run("returns ErrUnknownLanguage for a language that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			a := answers()
			a.Languages = []workspace.Language{"cobol"}
			_, err := repository(t, directory(t)).New(a, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrUnknownLanguage, "New")
			assert.Contains(t, err.Error(), "alpha, beta, gamma", "the languages that the error lists")
		})

		t.Run("returns ErrLanguagePresent for a language that the answers name twice", func(t *testing.T) {
			t.Parallel()
			a := answers()
			a.Languages = []workspace.Language{alpha, alpha}
			_, err := repository(t, directory(t)).New(a, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrLanguagePresent, "New")
		})

		t.Run("returns ErrUnsupported for a language without a producer", func(t *testing.T) {
			t.Parallel()
			a := answers()
			a.Languages = []workspace.Language{gamma}
			_, err := repository(t, directory(t)).New(a, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrUnsupported, "New")
		})

		t.Run("returns the error of a producer", func(t *testing.T) {
			t.Parallel()
			failing := baseline.Producer{Name: "failing", Producer: producer{templates: fstest.MapFS{}, err: errRender}}
			r, err := baseline.Open(directory(t), catalog(t), version, failing)
			assert.NoError(t, err, "Open")
			_, err = r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, errRender, "New")
		})

		t.Run(
			"returns ErrInvalidContribution of the rendering for a contribution that is not valid",
			func(t *testing.T) {
				t.Parallel()
				odd := baseline.Producer{Name: "odd", Producer: contributing{
					producer: producer{templates: fstest.MapFS{}},
					part:     workflow.Contribution{Jobs: []workflow.Job{{ID: "check"}}},
				}}
				r, err := baseline.Open(directory(t), catalog(t), version, odd)
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.ErrorIs(t, err, render.ErrInvalidContribution, "New")
			},
		)

		t.Run("returns ErrDefect of the options for options that do not encode", func(t *testing.T) {
			t.Parallel()
			odd := baseline.Producer{Name: "odd", Producer: unwritable{producer: producer{templates: fstest.MapFS{}}}}
			r, err := baseline.Open(directory(t), catalog(t), version, odd)
			assert.NoError(t, err, "Open")
			_, err = r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, options.ErrDefect, "New")
		})

		t.Run("returns ErrInvalidFile for a producer that renders .ergon.yaml", func(t *testing.T) {
			t.Parallel()
			r, err := baseline.Open(directory(t), catalog(t), version, templated("odd", fstest.MapFS{
				"managed/.ergon.yaml.tmpl": {Data: []byte("name: demo\n")},
			}))
			assert.NoError(t, err, "Open")
			_, err = r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrInvalidFile, "New")
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the files of the language and its fragments", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			changes, err := r.Add([]workspace.Language{beta}, baseline.Options{})
			assert.NoError(t, err, "Add of beta")
			assert.Equal(t, paths(changes), []string{ignore, betaCfg, lockPath}, "the files that Add wrote")
			assert.Equal(t, content(t, root, ignore), "# common\nalpha/\nbeta/\n", "the .gitignore")
		})

		t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			_, err := repository(t, directory(t)).Add([]workspace.Language{beta}, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Add")
		})

		t.Run("returns ErrLanguagePresent for a language that the answers have", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.Add([]workspace.Language{alpha}, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrLanguagePresent, "Add of alpha")
		})

		t.Run("returns ErrConflict for an edited file that it would change", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, ignore, "# edited by hand\n")
			_, err := r.Add([]workspace.Language{beta}, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrConflict, "Add of beta")
			assert.False(t, exists(t, root, betaCfg), "the file of beta after the conflict")
		})

		t.Run("overwrites an edited file with Force", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, ignore, "# edited by hand\n")
			_, err := r.Add([]workspace.Language{beta}, baseline.Options{Force: true})
			assert.NoError(t, err, "Add of beta with Force")
			assert.Equal(t, content(t, root, ignore), "# common\nalpha/\nbeta/\n", "the .gitignore")
		})
	})

	t.Run("Remove", func(t *testing.T) {
		t.Parallel()

		t.Run("removes the files and the section of the language and rewrites its shared files", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			changes, err := r.Remove([]workspace.Language{alpha})
			assert.NoError(t, err, "Remove of alpha")
			assert.Equal(t, changes, []baseline.Change{
				{Path: config, Action: baseline.Wrote},
				{Path: ignore, Action: baseline.Wrote},
				{Path: alphaCfg, Action: baseline.Removed},
				{Path: lockPath, Action: baseline.Wrote},
			}, "the changes of Remove")
			assert.Equal(t, content(t, root, ignore), "# common\n", "the .gitignore")
			assert.Equal(t, content(t, root, config), "common:\n  # The greeting of greeting.txt.\n  greeting: hello\n",
				"the .ergon.yaml")
		})

		t.Run("keeps the files of a shared toolchain while one of its languages remains", func(t *testing.T) {
			t.Parallel()
			r, root := sharedRepository(t)
			changes, err := r.Remove([]workspace.Language{delta})
			assert.NoError(t, err, "Remove of delta")
			assert.Equal(t, paths(changes), []string{ignore, "delta.txt", lockPath}, "the files that Remove changed")
			assert.True(t, exists(t, root, sharedFile), "the file of the toolchain")
		})

		t.Run("removes the files of a shared toolchain with its last language", func(t *testing.T) {
			t.Parallel()
			r, root := sharedRepository(t)
			_, err := r.Remove([]workspace.Language{delta, epsilon})
			assert.NoError(t, err, "Remove of delta and epsilon")
			assert.False(t, exists(t, root, sharedFile), "the file of the toolchain")
			assert.Equal(t, content(t, root, ignore), "# common\n", "the .gitignore")
		})

		t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			_, err := repository(t, directory(t)).Remove([]workspace.Language{alpha})
			assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Remove")
		})

		t.Run("returns ErrUnknownLanguage for a language that the catalog does not have", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.Remove([]workspace.Language{"cobol"})
			assert.ErrorIs(t, err, baseline.ErrUnknownLanguage, "Remove of cobol")
			assert.Contains(t, err.Error(), "alpha, beta, gamma", "the languages that the error lists")
		})

		t.Run("returns ErrLanguageAbsent for a language that the answers do not have", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.Remove([]workspace.Language{beta})
			assert.ErrorIs(t, err, baseline.ErrLanguageAbsent, "Remove of beta")
		})

		t.Run("returns ErrConflict for an edited file of the language", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, alphaCfg, "alpha, edited\n")
			_, err := r.Remove([]workspace.Language{alpha})
			assert.ErrorIs(t, err, baseline.ErrConflict, "Remove of alpha")
			assert.Equal(t, content(t, root, ignore), "# common\nalpha/\n", "the .gitignore after the conflict")
		})
	})

	t.Run("Sync", func(t *testing.T) {
		t.Parallel()

		t.Run("rewrites the files that a changed answer renders", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			changes, err := r.Sync(func(a *language.Answers) { a.Owner = "Other B.V." }, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Equal(t, paths(changes), []string{license, lockPath}, "the files that Sync wrote")
			assert.Equal(t, content(t, root, license), "Copyright Other B.V.\n", "the LICENSE")
		})

		t.Run("writes nothing for a repository at the baseline", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			changes, err := r.Sync(nil, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Empty(t, changes, "the changes of Sync")
		})

		t.Run("writes a missing managed file", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
			changes, err := r.Sync(nil, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Equal(t, paths(changes), []string{license}, "the files that Sync wrote")
		})

		t.Run("moves an option at the previous baseline to the baseline of the installed ergon", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			r, err := baseline.Open(root, catalog(t), version, common("hi"))
			assert.NoError(t, err, "Open of the later baseline")
			changes, err := r.Sync(nil, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Equal(t, paths(changes), []string{config, greeting, lockPath}, "the files that Sync wrote")
			assert.Equal(t, content(t, root, greeting), "hi\n", "the greeting")
			assert.Contains(t, content(t, root, lockPath), `"common.greeting": "hi"`, "the lock")
		})

		t.Run("keeps an option that the repository changed", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			put(t, root, config, "common:\n  greeting: hey\nalpha:\n  greeting: hi\n")
			r, err := baseline.Open(root, catalog(t), version, common("hi"))
			assert.NoError(t, err, "Open of the later baseline")
			_, err = r.Sync(nil, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Equal(t, content(t, root, greeting), "hey\n", "the greeting")
		})

		t.Run("writes the answer that update sets into a key that states the answer of the lock", func(t *testing.T) {
			t.Parallel()
			r, root := owned(t)
			_, err := r.Sync(func(a *language.Answers) { a.Owner = "Other B.V." }, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.Contains(t, content(t, root, config), "owner: Other B.V.\n", "the .ergon.yaml")
		})

		t.Run("returns ErrInvalid of the options for a key that states neither answer", func(t *testing.T) {
			t.Parallel()
			r, root := owned(t)
			put(t, root, config, edited)
			_, err := r.Sync(func(a *language.Answers) { a.Owner = "Other B.V." }, baseline.Options{})
			assert.ErrorIs(t, err, options.ErrInvalid, "Sync")
			assert.Contains(t, err.Error(), editedKey, "the error")
		})

		t.Run("writes the other files and returns ErrConflict for an edited file", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, ignore, "# edited by hand\n")
			changes, err := r.Sync(func(a *language.Answers) { a.Owner = "Other B.V." }, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrConflict, "Sync")
			assert.Equal(t, paths(changes), []string{license, lockPath}, "the files that Sync wrote")
			assert.Equal(t, content(t, root, ignore), "# edited by hand\n", "the edited .gitignore")
		})

		t.Run("overwrites an edited file with Force", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, ignore, "# edited by hand\n")
			_, err := r.Sync(nil, baseline.Options{Force: true})
			assert.NoError(t, err, "Sync with Force")
			assert.Equal(t, content(t, root, ignore), "# common\nalpha/\n", "the .gitignore")
		})

		t.Run("removes the files of a language that the answers no longer have", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			_, err := r.Sync(func(a *language.Answers) { a.Languages = nil }, baseline.Options{})
			assert.NoError(t, err, "Sync")
			assert.False(t, exists(t, root, alphaCfg), "the file of alpha")
		})

		t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			_, err := repository(t, directory(t)).Sync(nil, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Sync")
		})

		t.Run("returns ErrUnknownLanguage for a changed answer that does not render", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.Sync(
				func(a *language.Answers) { a.Languages = []workspace.Language{"cobol"} },
				baseline.Options{},
			)
			assert.ErrorIs(t, err, baseline.ErrUnknownLanguage, "Sync")
		})
	})

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no finding for a repository at the baseline", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			findings, err := r.Check()
			assert.NoError(t, err, "Check")
			assert.Empty(t, findings, "the findings")
		})

		t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			_, err := repository(t, directory(t)).Check()
			assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Check")
		})

		t.Run("returns ErrInvalid of the options for .ergon.yaml that the producers do not accept", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, config, "common:\n  greeting: [a]\n")
			_, err := r.Check()
			assert.ErrorIs(t, err, options.ErrInvalid, "Check")
		})

		t.Run("returns ErrInvalid of the options for a key that differs from the answer", func(t *testing.T) {
			t.Parallel()
			r, root := owned(t)
			put(t, root, config, edited)
			_, err := r.Check()
			assert.ErrorIs(t, err, options.ErrInvalid, "Check")
			assert.Contains(t, err.Error(), editedKey, "the error")
		})

		t.Run("returns ErrUnsupported for a lock with a language that lost its producer", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			c := new(language.Catalog)
			assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain")
			assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register")
			r, err := baseline.Open(root, c, version, common("hello"))
			assert.NoError(t, err, "Open")
			_, err = r.Check()
			assert.ErrorIs(t, err, baseline.ErrUnsupported, "Check")
		})
	})

	t.Run("Options", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the options of a base producer as the section states them", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, config, "common:\n  greeting: welcome\n")
			got, err := r.Options("common")
			assert.NoError(t, err, "Options")
			assert.Equal(t, got, language.Options(&greetings{Greeting: "welcome"}), "the options")
		})

		t.Run("returns the options of a language at the baseline", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			got, err := r.Options(string(alpha))
			assert.NoError(t, err, "Options")
			assert.Equal(t, got, language.Options(&greetings{Greeting: "hi"}), "the options")
		})

		t.Run("writes nothing", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, config, "common:\n  greeting: welcome\n")
			_, err := r.Options("common")
			assert.NoError(t, err, "Options")
			assert.Equal(t, content(t, root, config), "common:\n  greeting: welcome\n", "the content of .ergon.yaml")
		})

		t.Run("returns ErrUnknownSection for a producer without options", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.Options("license")
			assert.ErrorIs(t, err, baseline.ErrUnknownSection, "Options")
		})

		t.Run("returns ErrNotInitialized for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			_, err := repository(t, directory(t)).Options("common")
			assert.ErrorIs(t, err, baseline.ErrNotInitialized, "Options")
		})

		t.Run("returns ErrUnsupported for a lock with a language that lost its producer", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			c := new(language.Catalog)
			assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain")
			assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register")
			r, err := baseline.Open(root, c, version, common("hello"))
			assert.NoError(t, err, "Open")
			_, err = r.Options("common")
			assert.ErrorIs(t, err, baseline.ErrUnsupported, "Options")
		})

		t.Run("returns ErrInvalid of the options for .ergon.yaml that the producers do not accept", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, config, "common:\n  greeting: [a]\n")
			_, err := r.Options("common")
			assert.ErrorIs(t, err, options.ErrInvalid, "Options")
		})

		t.Run("returns ErrInvalid of the options for a key that differs from the answer", func(t *testing.T) {
			t.Parallel()
			r, root := owned(t)
			put(t, root, config, edited)
			_, err := r.Options(ownerSection)
			assert.ErrorIs(t, err, options.ErrInvalid, "Options")
			assert.Contains(t, err.Error(), editedKey, "the error")
		})
	})
}

// common returns the base producer of the cases: a shared, a managed, a seeded and a YAML file,
// and greeting.txt, which renders its option at the baseline greeting. The LICENSE renders the
// owner, so an answer can change it.
func common(greeting string) baseline.Producer {
	return baseline.Producer{Name: "common", Producer: configurable{greeting: greeting, producer: producer{
		templates: fstest.MapFS{
			"shared/.gitignore.tmpl":      {Data: []byte("# common\n")},
			"managed/LICENSE.tmpl":        {Data: []byte("Copyright {{% .Answers.Owner %}}\n")},
			"seeded/README.md.tmpl":       {Data: []byte("# {{% .Answers.Name %}}\n")},
			"managed/.github/ci.yml.tmpl": {Data: []byte(workflowContent)},
			"managed/greeting.txt.tmpl":   {Data: []byte("{{% .Options.Greeting %}}\n")},
		},
	}}}
}

// templated returns a base producer of the cases named name, whose templates are templates.
func templated(name string, templates fstest.MapFS) baseline.Producer {
	return baseline.Producer{Name: name, Producer: producer{templates: templates}}
}

// files returns the producer of a language of the cases, which contributes a fragment of
// .gitignore and renders its own file.
func files(fragment, name, content string) producer {
	return producer{templates: fstest.MapFS{
		"shared/.gitignore.tmpl":    {Data: []byte(fragment)},
		"managed/" + name + ".tmpl": {Data: []byte(content)},
	}}
}

// answers returns new answers of the cases, with the language alpha, which a case may change.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Languages:       []workspace.Language{alpha},
		Owner:           "Dokimasia B.V.",
		License:         spdx.MIT,
		Year:            2026,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
	}
}

// catalog returns the catalog of the cases: alpha with options, beta without options, and gamma
// without a producer.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain of tool")
	assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool},
		configurable{producer: files("alpha/\n", alphaCfg, "alpha\n"), greeting: "hi"}), "Register of alpha")
	assert.NoError(t, language.Register(c, language.Declaration{Name: beta, Toolchain: tool},
		files("beta/\n", betaCfg, "beta\n")), "Register of beta")
	assert.NoError(t, language.Register(c, language.Declaration{Name: gamma, Toolchain: tool}), "Register of gamma")
	return c
}

// sharedCatalog returns the catalog of the toolchain shared, which renders sharedFile and a
// fragment of .gitignore, and of its languages delta and epsilon.
func sharedCatalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: shared},
		files("shared/\n", sharedFile, "shared\n")), "RegisterToolchain of shared")
	assert.NoError(t, language.Register(c, language.Declaration{Name: delta, Toolchain: shared},
		files("delta/\n", "delta.txt", "delta\n")), "Register of delta")
	assert.NoError(t, language.Register(c, language.Declaration{Name: epsilon, Toolchain: shared},
		files("epsilon/\n", "epsilon.txt", "epsilon\n")), "Register of epsilon")
	return c
}

// sharedRepository returns a repository and its directory after New with delta and epsilon.
func sharedRepository(t *testing.T) (*baseline.Repository, *os.Root) {
	t.Helper()
	root := directory(t)
	r, err := baseline.Open(root, sharedCatalog(t), version, common("hello"))
	assert.NoError(t, err, "Open")
	a := answers()
	a.Languages = []workspace.Language{delta, epsilon}
	_, err = r.New(a, baseline.Options{})
	assert.NoError(t, err, "New with delta and epsilon")
	return r, root
}

// directory returns an os.Root of a new temporary directory, which the test closes.
func directory(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	assert.NoError(t, err, "OpenRoot of the temporary directory")
	t.Cleanup(func() { _ = root.Close() })
	return root
}

// repository returns the repository of the cases on fsys, with the producer common at the
// greeting hello.
func repository(t *testing.T, fsys baseline.FS) *baseline.Repository {
	t.Helper()
	r, err := baseline.Open(fsys, catalog(t), version, common("hello"))
	assert.NoError(t, err, "Open of the repository")
	return r
}

// initialized returns a repository and its directory after New with the answers of the cases.
func initialized(t *testing.T) (*baseline.Repository, *os.Root) {
	t.Helper()
	root := directory(t)
	r := repository(t, root)
	_, err := r.New(answers(), baseline.Options{})
	assert.NoError(t, err, "New of the repository")
	return r, root
}

// owned returns a repository and its directory after New with the answers of the cases, whose base
// producers are common and owner.
func owned(t *testing.T) (*baseline.Repository, *os.Root) {
	t.Helper()
	root := directory(t)
	r, err := baseline.Open(root, catalog(t), version, common("hello"),
		baseline.Producer{Name: ownerSection, Producer: owner{}})
	assert.NoError(t, err, "Open with the producer owner")
	_, err = r.New(answers(), baseline.Options{})
	assert.NoError(t, err, "New of the repository")
	return r, root
}

// content returns the content of the file name in root.
func content(t *testing.T, root *os.Root, name string) string {
	t.Helper()
	data, err := root.ReadFile(name)
	assert.NoError(t, err, "ReadFile of "+name)
	return string(data)
}

// exists reports whether root has the file name.
func exists(t *testing.T, root *os.Root, name string) bool {
	t.Helper()
	_, err := root.Stat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	assert.NoError(t, err, "Stat of "+name)
	return true
}

// put writes content to the file name in root, with its directories.
func put(t *testing.T, root *os.Root, name, content string) {
	t.Helper()
	assert.NoError(t, root.MkdirAll(path.Dir(name), 0o755), "MkdirAll of the directory of "+name)
	assert.NoError(t, root.WriteFile(name, []byte(content), 0o644), "WriteFile of "+name)
}

// paths returns the paths of changes, in their order.
func paths(changes []baseline.Change) []string {
	out := make([]string, 0, len(changes))
	for _, c := range changes {
		out = append(out, c.Path)
	}
	return out
}

// sum returns the SHA-256 digest of s as 64 lowercase hexadecimal digits.
func sum(s string) string {
	digest := sha256.Sum256([]byte(s))
	return hex.EncodeToString(digest[:])
}
