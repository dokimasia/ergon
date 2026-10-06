// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
)

// The toolchains and the languages of the cases. alpha and beta render files, and gamma registers
// no initializer. The toolchain shared renders a file that its languages delta and epsilon share.
const (
	tool    workspace.Toolchain = "tool"
	alpha   workspace.Language  = "alpha"
	beta    workspace.Language  = "beta"
	gamma   workspace.Language  = "gamma"
	shared  workspace.Toolchain = "shared"
	delta   workspace.Language  = "delta"
	epsilon workspace.Language  = "epsilon"
)

// The paths that the producers of the cases render, and the path of the lock.
const (
	ignore   = ".gitignore"
	license  = "LICENSE"
	readme   = "README.md"
	config   = ".ergon.yaml"
	workflow = ".github/ci.yml"
	alphaCfg = "alpha/settings.txt"
	betaCfg  = "beta.txt"
	lockPath = ".ergon/init.lock"

	// sharedFile is the file of the toolchain shared.
	sharedFile = "shared.txt"
)

// version is the version of ergon that the locks of the cases record.
const version = "1.2.3"

// errRender is the error of a producer that cannot render its answers.
var errRender = errors.New("render: the answers do not render")

// renderer is an initializer whose function renders the files.
type renderer func(a *language.Answers) ([]language.File, error)

// Files returns the files of f for a.
func (f renderer) Files(a *language.Answers) ([]language.File, error) {
	return f(a)
}

// common is the base producer of the cases: a shared, a managed, a seeded, a configured and a
// YAML file. The LICENSE renders the owner, so an answer can change it.
var common = baseline.Producer{
	Name: "common",
	Initializer: renderer(func(a *language.Answers) ([]language.File, error) {
		return []language.File{
			{Path: ignore, Class: language.Managed, Fragment: []byte("# common\n")},
			{Path: license, Class: language.Managed, Content: []byte("Copyright " + a.Owner + "\n")},
			{Path: readme, Class: language.Seeded, Content: []byte("# " + a.Name + "\n")},
			{Path: config, Class: language.Configured, Content: []byte("name: " + a.Name + "\n")},
			{Path: workflow, Class: language.Managed, Content: []byte(workflowContent)},
		}, nil
	}),
}

// workflowContent is the managed YAML file of common: a comment, a scalar, a map and a list.
const workflowContent = "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n"

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
				name: "returns ErrInvalidOpen for a producer without an initializer", fsys: directory(t),
				catalog: catalog(t), version: version, producers: []baseline.Producer{{Name: "common"}},
			},
			{
				name: "returns ErrInvalidOpen for a producer with an invalid name", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: "Common", Initializer: common.Initializer}},
			},
			{
				name: "returns ErrInvalidOpen for a producer named as a language", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: string(alpha), Initializer: common.Initializer}},
			},
			{
				name: "returns ErrInvalidOpen for a producer named as a toolchain", fsys: directory(t),
				catalog: catalog(t), version: version,
				producers: []baseline.Producer{{Name: string(tool), Initializer: common.Initializer}},
			},
			{
				name: "returns ErrInvalidOpen for two producers with one name", fsys: directory(t),
				catalog: catalog(t), version: version, producers: []baseline.Producer{common, common},
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

		t.Run("writes the managed, seeded and configured files and the lock", func(t *testing.T) {
			t.Parallel()
			root := directory(t)
			changes, err := repository(t, root).New(answers(), baseline.Options{})
			assert.NoError(t, err, "New")
			assert.Equal(t, paths(changes), []string{config, workflow, ignore, license, readme, alphaCfg, lockPath},
				"the files that New wrote")
			assert.Equal(t, content(t, root, readme), "# demo\n", "the seeded README.md")
			assert.Equal(t, content(t, root, config), "name: demo\n", "the configured .ergon.yaml")
			assert.Equal(t, content(t, root, alphaCfg), "alpha\n", "the file of alpha")
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
			r, err := baseline.Open(root, sharedCatalog(t), version, common)
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

		t.Run("returns ErrInitialized for a repository with a lock", func(t *testing.T) {
			t.Parallel()
			r, _ := initialized(t)
			_, err := r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrInitialized, "a second New")
		})

		t.Run("returns ErrInvalidLock for a repository with a lock that does not parse", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			put(t, root, lockPath, "not JSON\n")
			_, err := r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrInvalidLock, "New over a lock that does not parse")
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

		t.Run("returns ErrUnsupported for a language without an initializer", func(t *testing.T) {
			t.Parallel()
			a := answers()
			a.Languages = []workspace.Language{gamma}
			_, err := repository(t, directory(t)).New(a, baseline.Options{})
			assert.ErrorIs(t, err, baseline.ErrUnsupported, "New")
		})

		t.Run("returns the error of a producer", func(t *testing.T) {
			t.Parallel()
			failing := baseline.Producer{
				Name: "failing",
				Initializer: renderer(func(*language.Answers) ([]language.File, error) {
					return nil, errRender
				}),
			}
			r, err := baseline.Open(directory(t), catalog(t), version, failing)
			assert.NoError(t, err, "Open")
			_, err = r.New(answers(), baseline.Options{})
			assert.ErrorIs(t, err, errRender, "New")
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

		t.Run("removes the files of the language and rewrites its shared files", func(t *testing.T) {
			t.Parallel()
			r, root := initialized(t)
			changes, err := r.Remove([]workspace.Language{alpha})
			assert.NoError(t, err, "Remove of alpha")
			assert.Equal(t, changes, []baseline.Change{
				{Path: ignore, Action: baseline.Wrote},
				{Path: alphaCfg, Action: baseline.Removed},
				{Path: lockPath, Action: baseline.Wrote},
			}, "the changes of Remove")
			assert.Equal(t, content(t, root, ignore), "# common\n", "the .gitignore")
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

		t.Run("returns ErrUnsupported for a lock with a language that lost its initializer", func(t *testing.T) {
			t.Parallel()
			_, root := initialized(t)
			c := new(language.Catalog)
			assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain")
			assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}), "Register")
			r, err := baseline.Open(root, c, version, common)
			assert.NoError(t, err, "Open")
			_, err = r.Check()
			assert.ErrorIs(t, err, baseline.ErrUnsupported, "Check")
		})
	})
}

// languageFiles returns the initializer of a language that contributes a fragment of .gitignore
// and renders its own file.
func languageFiles(fragment, name, content string) language.Initializer {
	return renderer(func(*language.Answers) ([]language.File, error) {
		return []language.File{
			{Path: ignore, Class: language.Managed, Fragment: []byte(fragment)},
			{Path: name, Class: language.Managed, Content: []byte(content)},
		}, nil
	})
}

// answers returns new answers of the cases, with the language alpha, which a case may change.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Languages:       []workspace.Language{alpha},
		Owner:           "Dokimasia B.V.",
		License:         "MIT",
		Year:            2026,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
	}
}

// catalog returns the catalog of the cases: alpha and beta with an initializer, and gamma
// without one.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: tool}), "RegisterToolchain of tool")
	assert.NoError(t, language.Register(c, language.Declaration{Name: alpha, Toolchain: tool},
		languageFiles("alpha/\n", alphaCfg, "alpha\n")), "Register of alpha")
	assert.NoError(t, language.Register(c, language.Declaration{Name: beta, Toolchain: tool},
		languageFiles("beta/\n", betaCfg, "beta\n")), "Register of beta")
	assert.NoError(t, language.Register(c, language.Declaration{Name: gamma, Toolchain: tool}), "Register of gamma")
	return c
}

// sharedCatalog returns the catalog of the toolchain shared, which renders sharedFile and a
// fragment of .gitignore, and of its languages delta and epsilon.
func sharedCatalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, language.RegisterToolchain(c, language.Toolchain{Name: shared},
		languageFiles("shared/\n", sharedFile, "shared\n")), "RegisterToolchain of shared")
	assert.NoError(t, language.Register(c, language.Declaration{Name: delta, Toolchain: shared},
		languageFiles("delta/\n", "delta.txt", "delta\n")), "Register of delta")
	assert.NoError(t, language.Register(c, language.Declaration{Name: epsilon, Toolchain: shared},
		languageFiles("epsilon/\n", "epsilon.txt", "epsilon\n")), "Register of epsilon")
	return c
}

// sharedRepository returns a repository and its directory after New with delta and epsilon.
func sharedRepository(t *testing.T) (*baseline.Repository, *os.Root) {
	t.Helper()
	root := directory(t)
	r, err := baseline.Open(root, sharedCatalog(t), version, common)
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

// repository returns the repository of the cases on fsys, with the producer common.
func repository(t *testing.T, fsys baseline.FS) *baseline.Repository {
	t.Helper()
	r, err := baseline.Open(fsys, catalog(t), version, common)
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
