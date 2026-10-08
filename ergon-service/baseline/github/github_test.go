// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github_test

import (
	"io/fs"
	"os"
	"path"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
)

// name pins the name of the producer of the GitHub files.
const name = "github"

// The workflows and the configuration of Dependabot that the contributions of the cases change.
const (
	ciPath         = ".github/workflows/ci.yml"
	releasePath    = ".github/workflows/release.yml"
	securityPath   = ".github/workflows/security.yml"
	dependabotPath = ".github/dependabot.yml"
)

// setupAlpha is the action that sets up the toolchain of the cases.
var setupAlpha = workflow.Action{
	Uses:    "alpha-lang/setup-alpha",
	Commit:  "0123456789abcdef0123456789abcdef01234567",
	Release: "v1.2.3",
}

// read is the permission of a job to read the contents of the repository.
var read = map[string]string{"contents": "read"}

// part is a producer of the cases without templates, with a part of the workflows.
type part struct {
	// contribution is the part that Contribution returns.
	contribution workflow.Contribution
}

// Templates returns no template.
func (part) Templates() fs.FS {
	return fstest.MapFS{}
}

// Contribution returns p.contribution, whatever the options.
func (p part) Contribution(language.Options) workflow.Contribution {
	return p.contribution
}

func TestGithub(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is github", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, github.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the GitHub files at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer())
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run(
				"renders the jobs, the release steps, the analyses and the updates of every producer",
				func(t *testing.T) {
					t.Parallel()
					dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer(),
						baseline.Producer{Name: "tool", Producer: part{contribution: toolchain()}},
						baseline.Producer{Name: "alpha", Producer: part{contribution: languages()}},
					)
					baselinetest.Hygiene(t, dir)
					for _, file := range []string{ciPath, releasePath, securityPath, dependabotPath} {
						got, err := os.ReadFile(path.Join(dir, file))
						assert.NoError(t, err, "ReadFile of "+file)
						golden.Match(t, path.Join("contributions", path.Base(file)), got, golden.ShouldUpdate())
					}
				},
			)

			t.Run("renders no configuration of Dependabot without an update", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer())
				_, err := os.Stat(path.Join(dir, dependabotPath))
				assert.ErrorIs(t, err, fs.ErrNotExist, "Stat of "+dependabotPath)
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := github.Producer{}.Options().(*github.Options)
				second, _ := github.Producer{}.Options().(*github.Options)
				first.Runners[0] = "changed"
				assert.Equal(t, second.Runners[0], "ubuntu-26.04", "the first runner of the second value")
			})
		})

		t.Run("Data", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the jobs of the contributions for the options", func(t *testing.T) {
				t.Parallel()
				o, _ := github.Producer{}.Options().(*github.Options)
				o.Runners = option.Runners{"ubuntu-26.04", "windows-2025"}
				c := languages()
				want, err := o.Jobs(&c)
				assert.NoError(t, err, "Jobs")
				got, err := github.Producer{}.Data(baselinetest.Answers(), o, &c)
				assert.NoError(t, err, "Data")
				w, ok := got.(github.Workflows)
				assert.True(t, ok, "the data is a Workflows")
				assert.Equal(t, w.Jobs, want, "the jobs")
			})

			t.Run(
				"returns a limit of wait that adds the limit of the section to the longest limit of a job",
				func(t *testing.T) {
					t.Parallel()
					o, _ := github.Producer{}.Options().(*github.Options)
					o.CI.Timeout = 25
					c := languages()
					got, err := github.Producer{}.Data(baselinetest.Answers(), o, &c)
					assert.NoError(t, err, "Data")
					w, ok := got.(github.Workflows)
					assert.True(t, ok, "the data is a Workflows")
					assert.Equal(t, w.Wait, 30, "the limit of wait, which is the 5 minutes of check-beta plus 25")
				},
			)

			t.Run("returns the data of the options at the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				c := languages()
				want, err := github.Producer{}.Data(baselinetest.Answers(), github.Producer{}.Options(), &c)
				assert.NoError(t, err, "Data for the options at the baseline")
				got, err := github.Producer{}.Data(baselinetest.Answers(), nil, &c)
				assert.NoError(t, err, "Data")
				assert.Equal(t, got, want, "the data")
			})

			t.Run("returns ErrInvalid for a job on a runner that the section does not list", func(t *testing.T) {
				t.Parallel()
				c := workflow.Contribution{Jobs: []workflow.Job{{
					ID:    "check-alpha",
					Name:  "Alpha",
					Setup: &workflow.Setup{Runners: []string{"ubuntu-24.04"}, Timeout: 5},
					Steps: []workflow.Step{{Run: []string{"make check-alpha"}}},
				}}}
				_, err := github.Producer{}.Data(baselinetest.Answers(), nil, &c)
				assert.ErrorIs(t, err, option.ErrInvalid, "Data")
			})

			t.Run(
				"returns ErrInvalid from New for a job on a runner that the section does not list",
				func(t *testing.T) {
					t.Parallel()
					root, err := os.OpenRoot(t.TempDir())
					assert.NoError(t, err, "OpenRoot")
					t.Cleanup(func() { _ = root.Close() })
					c := workflow.Contribution{Setup: &workflow.Setup{Runners: []string{"ubuntu-24.04"}, Timeout: 5}}
					r, err := baseline.Open(root, new(language.Catalog), baselinetest.Version, producer(),
						baseline.Producer{Name: "tool", Producer: part{contribution: c}},
						baseline.Producer{Name: "alpha", Producer: part{contribution: languages()}},
					)
					assert.NoError(t, err, "Open")
					_, err = r.New(baselinetest.Answers(), baseline.Options{})
					assert.ErrorIs(t, err, option.ErrInvalid, "New")
				},
			)
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the job baseline, which runs ergon init check as a check of text", func(t *testing.T) {
				t.Parallel()
				got := github.Producer{}.Contribution(github.Producer{}.Options())
				assert.Equal(t, got, workflow.Contribution{Jobs: []workflow.Job{{
					ID:          "baseline",
					Name:        "Baseline",
					Text:        true,
					Timeout:     15,
					Permissions: read,
					Ergon:       true,
					Steps:       []workflow.Step{{Name: "Check the managed files", Run: []string{"ergon init check"}}},
				}}}, "the contribution")
			})

			t.Run("returns a valid contribution", func(t *testing.T) {
				t.Parallel()
				got := github.Producer{}.Contribution(github.Producer{}.Options())
				assert.NoError(t, got.Validate(), "Validate of the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				want := github.Producer{}.Contribution(github.Producer{}.Options())
				assert.Equal(t, github.Producer{}.Contribution(nil), want, "the contribution for nil options")
			})

			t.Run("returns the job with the timeout of the options", func(t *testing.T) {
				t.Parallel()
				o, _ := github.Producer{}.Options().(*github.Options)
				o.CI.Timeout = 25
				got := github.Producer{}.Contribution(o)
				assert.Equal(t, got.Jobs[0].Timeout, 25, "the timeout of baseline")
			})
		})
	})
}

// producer returns the producer of the GitHub files as a base producer.
func producer() baseline.Producer {
	return baseline.Producer{Name: github.Name, Producer: github.Producer{}}
}

// toolchain returns the contribution of the toolchain tool of the cases: the setup of alpha on
// every runner and two versions, its setup in a release, its analysis of CodeQL, and its updates.
func toolchain() workflow.Contribution {
	return workflow.Contribution{
		Setup: &workflow.Setup{
			Files:    "alpha.lock",
			Versions: []string{"1.0", "2.0"},
			Env:      map[string]string{"ALPHA_HOME": "/opt/alpha"},
			Steps: []workflow.Step{{
				Name: "Set up alpha",
				Uses: setupAlpha,
				With: map[string]string{"version": "${{ matrix.version }}", "cache": "false"},
			}},
			Timeout: 20,
		},
		Release: []workflow.Step{{
			Name: "Set up alpha",
			If:   "hashFiles('alpha.lock') != ''",
			Uses: setupAlpha,
			With: map[string]string{"version-file": ".alpha-version"},
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "alpha", Name: "Alpha", BuildMode: "none", Files: "alpha.lock", Timeout: 20},
		},
		Updates: []workflow.Update{{Ecosystem: "alpha", Directories: []string{"/", "/**/*"}}},
	}
}

// languages returns the contribution of the language alpha of the cases: a check that runs tools on
// the setup of the toolchain tool, a check on a setup of its own, a check of text that runs tools,
// and a job without a setup whose steps have each key of a step.
func languages() workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{
		{
			ID:          "check-alpha",
			Name:        "Alpha",
			Toolchain:   "tool",
			Permissions: read,
			Tools:       true,
			Steps: []workflow.Step{{
				Name: "Check alpha",
				If:   "github.event_name != 'schedule'",
				Run:  []string{"make check-alpha"},
			}},
		},
		{
			ID:          "check-beta",
			Name:        "Beta",
			Setup:       &workflow.Setup{Runners: []string{"ubuntu-26.04", "windows-2025"}, Timeout: 5},
			Permissions: read,
			Steps:       []workflow.Step{{Run: []string{"make check-beta"}}},
		},
		{
			ID:      "lint-text",
			Name:    "Text",
			If:      "github.event_name == 'pull_request'",
			Text:    true,
			Timeout: 3,
			History: true,
			Ergon:   true,
			Tools:   true,
			Steps: []workflow.Step{
				{Env: map[string]string{"BASE": "main"}, Run: []string{"ergon tool run tool.lint"}},
			},
		},
		{
			ID:          "steps",
			Name:        "Steps",
			Timeout:     4,
			Permissions: map[string]string{"contents": "read", "pull-requests": "write"},
			Steps: []workflow.Step{
				{ID: "pin", Run: []string{"echo version=1 >> \"$GITHUB_OUTPUT\""}},
				{If: "steps.pin.outputs.version == '1'", Uses: setupAlpha},
				{Uses: setupAlpha, With: map[string]string{"globs": "**/*.md\n#node_modules"}},
				{
					Name: "Report",
					Env:  map[string]string{"VERSION": "${{ steps.pin.outputs.version }}"},
					Run:  []string{"if [ -n \"$VERSION\" ]; then", "  echo \"$VERSION\"", "fi"},
				},
			},
		},
	}}
}
