// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
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
	nightlyPath    = ".github/workflows/nightly.yml"
	releasePath    = ".github/workflows/release.yml"
	versionPath    = ".github/workflows/version.yml"
	securityPath   = ".github/workflows/security.yml"
	codeqlPath     = ".github/workflows/codeql.yml"
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

// adjusted is the producer of the GitHub files whose options are the options at the baseline that
// its function changes.
type adjusted struct {
	github.Producer

	// adjust changes the options at the baseline.
	adjust func(*github.Options)
}

// Options returns the options of the GitHub files at the baseline, after the function of a changes
// them.
func (a adjusted) Options() language.Options {
	o, _ := a.Producer.Options().(*github.Options)
	a.adjust(o)
	return o
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
					files := []string{
						ciPath, nightlyPath, releasePath, versionPath, securityPath, codeqlPath, dependabotPath,
					}
					for _, file := range files {
						got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file)))
						assert.NoError(t, err, "ReadFile of "+file)
						golden.Match(t, path.Join("contributions", path.Base(file)), got, golden.ShouldUpdate())
					}
				},
			)

			t.Run("renders no configuration of Dependabot without an update", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer())
				files.Absent(
					t,
					filepath.Join(dir, filepath.FromSlash(dependabotPath)),
					"the configuration of Dependabot",
				)
			})

			t.Run("renders neither security.yml nor codeql.yml without GitHub Code Security", func(t *testing.T) {
				t.Parallel()
				private := adjusted{adjust: func(o *github.Options) { o.CodeSecurity = false }}
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(),
					baseline.Producer{Name: github.Name, Producer: private},
					baseline.Producer{Name: "tool", Producer: part{contribution: toolchain()}},
				)
				baselinetest.Hygiene(t, dir)
				files.Absent(t, filepath.Join(dir, filepath.FromSlash(securityPath)), "the workflow security.yml")
				files.Absent(t, filepath.Join(dir, filepath.FromSlash(codeqlPath)), "the workflow codeql.yml")
			})

			t.Run("renders the job prune-tools alone in nightly.yml without a nightly job", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer())
				got := files.Read(t, filepath.Join(dir, filepath.FromSlash(nightlyPath)))
				assert.HasSuffix(t, got, "\njobs:\n  prune-tools:\n    name: Prune the tool caches\n"+
					"    runs-on: ubuntu-26.04\n    timeout-minutes: 15\n    permissions:\n      actions: write\n"+
					"      contents: read\n    steps:\n"+
					"      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1\n"+
					"        with:\n          persist-credentials: false\n      - uses: ./.github/actions/setup-ergon\n"+
					"      - name: Delete the tool caches that newer caches replaced\n        env:\n"+
					"          GITHUB_TOKEN: \"${{ github.token }}\"\n        run: |\n          ergon tool ci prune\n",
					"the jobs of nightly.yml")
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
				nightly, err := o.NightlyJobs(&c)
				assert.NoError(t, err, "NightlyJobs")
				got, err := github.Producer{}.Data(baselinetest.Answers(), o, &c)
				assert.NoError(t, err, "Data")
				w, ok := got.(github.Workflows)
				assert.True(t, ok, "the data is a Workflows")
				expect.Equal(t, w.Jobs, want, "the jobs")
				expect.Equal(t, w.Nightly, nightly, "the nightly jobs")
				expect.True(t, w.Assets, "the assets")
				expect.Equal(t, w.TapOwner, "dokimasia", "the owner of the tap")
				expect.Equal(t, w.TapName, "homebrew-tap", "the name of the tap")
			})

			t.Run("returns the steps of each analysis of CodeQL under the condition of its language and its files",
				func(t *testing.T) {
					t.Parallel()
					c := toolchain()
					var got any
					var err error
					assert.Pure(t, func() []workflow.Step { return slices.Clone(c.CodeQL[0].Steps) }, func() {
						got, err = github.Producer{}.Data(baselinetest.Answers(), nil, &c)
					}, "the steps of the analysis of the contribution")
					assert.NoError(t, err, "Data")
					w, ok := got.(github.Workflows)
					assert.True(t, ok, "the data is a Workflows")
					want := slices.Clone(c.CodeQL[0].Steps)
					want[0].If = "inputs.language == 'alpha' && hashFiles(inputs.files) != ''"
					want[1].If = "inputs.language == 'alpha' && hashFiles(inputs.files) != '' && (runner.os == 'Linux')"
					assert.Equal(t, w.CodeQL, want, "the steps of codeql.yml")
				})

			t.Run("returns no assets and no tap for contributions without assets", func(t *testing.T) {
				t.Parallel()
				got, err := github.Producer{}.Data(baselinetest.Answers(), nil, &workflow.Contribution{})
				assert.NoError(t, err, "Data")
				assert.Equal(t, got, any(github.Workflows{}), "the data", assert.EquateEmpty())
			})

			t.Run("returns ErrInvalid for a nightly job on a runner that the section does not list",
				func(t *testing.T) {
					t.Parallel()
					c := workflow.Contribution{Nightly: []workflow.Job{{
						ID:    "fuzz-alpha",
						Name:  "Fuzz alpha",
						Setup: &workflow.Setup{Runners: []string{"ubuntu-24.04"}, Timeout: 120},
						Steps: []workflow.Step{{Run: []string{"make fuzz-alpha"}}},
					}}}
					_, err := github.Producer{}.Data(baselinetest.Answers(), nil, &c)
					assert.ErrorIs(t, err, option.ErrInvalid, "Data")
				})

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

			t.Run("returns the jobs baseline and workflows as checks of text", func(t *testing.T) {
				t.Parallel()
				got := github.Producer{}.Contribution(github.Producer{}.Options())
				assert.Equal(t, got, workflow.Contribution{Jobs: []workflow.Job{
					{
						ID:          "baseline",
						Name:        "Baseline",
						Text:        true,
						Timeout:     15,
						Permissions: read,
						Ergon:       true,
						Steps: []workflow.Step{
							{Name: "Check the managed files", Run: []string{"ergon init check"}},
						},
					},
					{
						ID:          "workflows",
						Name:        "Workflows",
						Text:        true,
						Timeout:     15,
						Permissions: read,
						Ergon:       true,
						Tools:       true,
						Steps:       []workflow.Step{{Name: "Lint the workflows", Run: []string{"make lint-github"}}},
					},
				}}, "the contribution")
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

			t.Run("returns the jobs with the timeout of the options", func(t *testing.T) {
				t.Parallel()
				o, _ := github.Producer{}.Options().(*github.Options)
				o.CI.Timeout = 25
				got := github.Producer{}.Contribution(o)
				assert.Length(t, got.Jobs, 2, "the jobs")
				expect.Equal(t, got.Jobs[0].Timeout, 25, "the timeout of baseline")
				expect.Equal(t, got.Jobs[1].Timeout, 25, "the timeout of workflows")
			})
		})

		t.Run("CheckLocal", func(t *testing.T) {
			t.Parallel()

			refused := []struct {
				name string
				give string
				want string
			}{
				{
					name: "returns ErrInvalidLocal for an update of github-actions",
					give: "updates:\n  - package-ecosystem: gomod\n  - package-ecosystem: github-actions\n",
					want: dependabotPath + " updates the ecosystem github-actions, so Dependabot would edit the " +
						"workflows and the actions under .github/, which ergon init renders",
				},
				{
					name: "returns ErrInvalidLocal for an update of pre-commit",
					give: "updates:\n  - package-ecosystem: pre-commit\n",
					want: dependabotPath + " updates the ecosystem pre-commit, so Dependabot would edit " +
						".pre-commit-config.yaml, which ergon init renders",
				},
				{
					name: "returns ErrInvalidLocal for updates that do not decode",
					give: "updates: 5\n",
					want: "the updates of " + dependabotPath,
				},
			}
			for _, tt := range refused {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					err := github.Producer{}.CheckLocal(dependabotPath, []byte(tt.give))
					assert.ErrorIs(t, err, language.ErrInvalidLocal, "CheckLocal")
					assert.Contains(t, err.Error(), tt.want, "the error")
				})
			}

			accepted := []struct {
				name string
				path string
				give string
			}{
				{
					name: "returns nil for updates of package managers",
					path: dependabotPath,
					give: "updates:\n  - package-ecosystem: gomod\n  - package-ecosystem: npm\n",
				},
				{
					name: "returns nil for another file",
					path: ciPath,
					give: "updates:\n  - package-ecosystem: github-actions\n",
				},
			}
			for _, tt := range accepted {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, github.Producer{}.CheckLocal(tt.path, []byte(tt.give)), "CheckLocal")
				})
			}

			t.Run("returns ErrInvalidLocal from New for a local file with an update of pre-commit", func(t *testing.T) {
				t.Parallel()
				root, err := os.OpenRoot(t.TempDir())
				assert.NoError(t, err, "OpenRoot")
				t.Cleanup(func() { _ = root.Close() })
				local := path.Join(".ergon/local", dependabotPath)
				assert.NoError(t, root.MkdirAll(path.Dir(local), 0o755), "MkdirAll of the directory of the local file")
				assert.NoError(t, root.WriteFile(local, []byte("updates:\n  - package-ecosystem: pre-commit\n"), 0o644),
					"WriteFile of the local file")
				r, err := baseline.Open(root, new(language.Catalog), baselinetest.Version, producer(),
					baseline.Producer{Name: "tool", Producer: part{contribution: toolchain()}},
					baseline.Producer{Name: "alpha", Producer: part{contribution: languages()}},
				)
				assert.NoError(t, err, "Open")
				_, err = r.New(baselinetest.Answers(), baseline.Options{})
				assert.ErrorIs(t, err, language.ErrInvalidLocal, "New")
			})
		})
	})
}

// producer returns the producer of the GitHub files as a base producer.
func producer() baseline.Producer {
	return baseline.Producer{Name: github.Name, Producer: github.Producer{}}
}

// toolchain returns the contribution of the toolchain tool of the cases: the setup of alpha on
// every runner and two versions, its setup in a release, its analysis of CodeQL with a step without
// a condition and a step with one before it, its setup and a list of its own packages before the
// dependency review, and its updates.
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
		CodeQL: []workflow.CodeQL{{
			Language:  "alpha",
			Name:      "Alpha",
			BuildMode: "none",
			Files:     "alpha.lock",
			Steps: []workflow.Step{
				{Name: "Set up alpha", Uses: setupAlpha, With: map[string]string{"version-file": ".alpha-version"}},
				{Name: "Name the modules of alpha", If: "runner.os == 'Linux'", Run: []string{"alpha modules"}},
			},
			Timeout: 20,
		}},
		Review: []workflow.Step{
			{
				Name: "Set up alpha",
				If:   "hashFiles('alpha.lock') != ''",
				Uses: setupAlpha,
				With: map[string]string{"version-file": ".alpha-version"},
			},
			{
				Name: "Leave the modules of alpha out of the license check",
				If:   "hashFiles('alpha.lock') != ''",
				Run:  []string{`echo "` + workflow.OwnPackages + `=pkg:alpha/../core" >> "$GITHUB_ENV"`},
			},
		},
		Updates: []workflow.Update{{Ecosystem: "alpha", Directories: []string{"/", "/**/*"}}},
	}
}

// languages returns the contribution of the language alpha of the cases: a check that runs tools on
// the setup of the toolchain tool, a check on a setup of its own, a check of text that runs tools,
// and a job without a setup whose steps have each key of a step. Its nightly jobs fuzz alpha on the
// setup of the toolchain tool, mutate beta on a setup of their own, and propose the updates of alpha
// with the token of the App. Its releases have assets with casks for the tap dokimasia/homebrew-tap.
func languages() workflow.Contribution {
	return workflow.Contribution{Assets: &workflow.Assets{Tap: "dokimasia/homebrew-tap"}, Nightly: []workflow.Job{
		{
			ID:          "fuzz-alpha",
			Name:        "Fuzz alpha",
			Toolchain:   "tool",
			Permissions: read,
			Tools:       true,
			Steps:       []workflow.Step{{Name: "Fuzz alpha", Run: []string{"make fuzz-alpha"}}},
		},
		{
			ID:          "mutate-beta",
			Name:        "Mutate beta",
			Setup:       &workflow.Setup{Files: "beta.lock", Timeout: 60},
			Permissions: read,
			Steps:       []workflow.Step{{Name: "Mutate beta", Run: []string{"make mutate-beta"}}},
		},
		{
			ID:          "propose-alpha",
			Name:        "Propose alpha",
			Timeout:     10,
			Permissions: map[string]string{"contents": "write", "pull-requests": "write"},
			Ergon:       true,
			App:         true,
			Steps: []workflow.Step{{
				Name: "Propose the updates of alpha",
				Env:  map[string]string{"GITHUB_TOKEN": workflow.AppToken},
				Run:  []string{`alpha propose --repo "$GITHUB_REPOSITORY"`},
			}},
		},
	}, Jobs: []workflow.Job{
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
