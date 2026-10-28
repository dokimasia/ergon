// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/terraform/workspace"
)

// The spellings of Terraform and of its toolchain in configuration, which the test pins.
const (
	pinnedLanguage  coreworkspace.Language  = "terraform"
	pinnedToolchain coreworkspace.Toolchain = "terraform"
)

// repositoryName is the name of the repository of the cases in the answers of its lock.
const repositoryName = "terraform-aws-vpc"

// The errors of a repository whose tags or whose lock cannot be read.
var (
	errTags    = errors.New("tags failed")
	errAnswers = errors.New("answers failed")
)

// terraformAnswers are the answers of the lock of a repository of Terraform.
var terraformAnswers = language.Answers{
	Name:      repositoryName,
	Languages: []coreworkspace.Language{"go", pinnedLanguage},
}

// tagged returns a function that returns tags as the tags of every repository.
func tagged(tags ...string) func(context.Context, string) (map[string]string, error) {
	return func(context.Context, string) (map[string]string, error) {
		out := make(map[string]string, len(tags))
		for _, name := range tags {
			out[name] = "0123456789abcdef0123456789abcdef01234567"
		}
		return out, nil
	}
}

// locked returns a function that returns a as the answers of the lock of every repository.
func locked(a language.Answers) func(context.Context, string) (language.Answers, error) {
	return func(context.Context, string) (language.Answers, error) {
		return a, nil
	}
}

func TestDiscoverer(t *testing.T) {
	t.Parallel()

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the repository as a package named after the repository", func(t *testing.T) {
			t.Parallel()
			d := workspace.Discoverer{Tags: tagged(), Answers: locked(terraformAnswers)}
			pkgs, err := d.Discover(t.Context(), t.TempDir())
			assert.NoError(t, err, "Discover")
			assert.Equal(t, pkgs, []coreworkspace.Package{{Name: repositoryName, Toolchain: pinnedToolchain, Dir: "."}},
				"the packages")
		})

		versions := []struct {
			tree files.Tree
			name string
			want string
			tags []string
		}{
			{
				name: "reads the version from the highest heading of the changelog",
				tree: files.Tree{"CHANGELOG.md": files.Text("# vpc\n\n## 1.2.0\n\n- b\n\n## 1.10.0\n\n- c\n")},
				want: "1.10.0",
			},
			{
				name: "reads the version from the highest tag v<version>",
				tags: []string{"v1.2.0", "v1.10.0", "vnext", "modules/one/v9.0.0", "demo@8.0.0"},
				want: "1.10.0",
			},
			{
				name: "takes the higher of the changelog and the tags",
				tree: files.Tree{"CHANGELOG.md": files.Text("# vpc\n\n## 0.4.0\n")},
				tags: []string{"v0.3.0"},
				want: "0.4.0",
			},
			{
				name: "takes the tag over a changelog without its version",
				tree: files.Tree{"CHANGELOG.md": files.Text("# vpc\n\n## 0.4.0\n")},
				tags: []string{"v0.5.0"},
				want: "0.5.0",
			},
			{
				name: "returns the zero version for a repository without a changelog and without tags",
				want: "0.0.0",
			},
		}
		for _, tt := range versions {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				d := workspace.Discoverer{Tags: tagged(tt.tags...), Answers: locked(terraformAnswers)}
				pkgs, err := d.Discover(t.Context(), files.Workspace(t, tt.tree))
				assert.NoError(t, err, "Discover")
				assert.Length(t, pkgs, 1, "the packages")
				assert.Equal(t, pkgs[0].Version, parse(t, tt.want), "the version")
			})
		}

		t.Run("returns no package for a lock without Terraform", func(t *testing.T) {
			t.Parallel()
			d := workspace.Discoverer{Tags: tagged(), Answers: locked(language.Answers{
				Name: repositoryName, Languages: []coreworkspace.Language{"go"},
			})}
			pkgs, err := d.Discover(t.Context(), t.TempDir())
			assert.NoError(t, err, "Discover")
			assert.Empty(t, pkgs, "the packages")
		})

		t.Run("returns no package for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			d := workspace.Discoverer{Tags: tagged(), Answers: locked(language.Answers{})}
			pkgs, err := d.Discover(t.Context(), t.TempDir())
			assert.NoError(t, err, "Discover")
			assert.Empty(t, pkgs, "the packages")
		})

		t.Run("returns the error of the answers", func(t *testing.T) {
			t.Parallel()
			failing := func(context.Context, string) (language.Answers, error) { return terraformAnswers, errAnswers }
			_, err := workspace.Discoverer{Tags: tagged(), Answers: failing}.Discover(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, errAnswers, "Discover")
		})

		t.Run("returns the error of the tags", func(t *testing.T) {
			t.Parallel()
			failing := func(context.Context, string) (map[string]string, error) { return nil, errTags }
			d := workspace.Discoverer{Tags: failing, Answers: locked(terraformAnswers)}
			_, err := d.Discover(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, errTags, "Discover")
		})

		t.Run("returns the error of a changelog that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"CHANGELOG.md/inner": files.Text("x\n")})
			d := workspace.Discoverer{Tags: tagged(), Answers: locked(terraformAnswers)}
			_, err := d.Discover(t.Context(), root)
			assert.HasError(t, err, "Discover")
			assert.Contains(t, err.Error(), "read CHANGELOG.md", "the error")
		})

		t.Run("returns the error of a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			absent := filepath.Join(t.TempDir(), "absent")
			d := workspace.Discoverer{Tags: tagged(), Answers: locked(terraformAnswers)}
			_, err := d.Discover(t.Context(), absent)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Discover")
		})
	})
}

// parse returns the version that s states, and stops the test tb for an s that version.Parse
// refuses.
func parse(tb testing.TB, s string) version.Version {
	tb.Helper()
	v, err := version.Parse(s)
	assert.NoError(tb, err, "Parse of "+s)
	return v
}
