// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/common"
)

// name pins the name of the producer of the common files.
const name = "common"

// The managed files whose commands the cases read: the configuration of pre-commit, and the
// Makefile.
const (
	preCommitConfig = ".pre-commit-config.yaml"
	makefile        = "Makefile"
)

// markdownlint is the pin of the action of markdownlint at the baseline.
var markdownlint = workflow.Action{
	Uses:    "DavidAnson/markdownlint-cli2-action",
	Commit:  "21c1be1b93ad9ed58fa840aacc3f279cde2a72ff",
	Release: "v24.2.0",
}

// read is the permission of the jobs of the common files.
var read = map[string]string{"contents": "read"}

// adjusted is the producer of the common files whose options are the options at the baseline that
// its function changes.
type adjusted struct {
	common.Producer

	// adjust changes the options at the baseline.
	adjust func(*common.Options)
}

// Options returns the options of the common files at the baseline, after the function of a changes
// them.
func (a adjusted) Options() language.Options {
	o, _ := a.Producer.Options().(*common.Options)
	a.adjust(o)
	return o
}

func TestCommon(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is common", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, common.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the common files at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), producer())
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			hooks := []struct {
				name   string
				golden string
				give   common.Hooks
			}{
				{
					name:   "renders a hook for each target of each stage in the order of the options",
					golden: "targets.yaml",
					give: common.Hooks{
						PreCommit: common.Targets{common.TargetFmt, common.TargetLint, common.TargetTest},
						PrePush:   common.Targets{common.TargetLint, common.TargetCheck},
					},
				},
				{
					name:   "renders no hook of a stage without targets",
					golden: "none.yaml",
					give:   common.Hooks{},
				},
				{
					name:   "renders the hooks of the stage pre-push alone",
					golden: "pre-push.yaml",
					give:   common.Hooks{PrePush: common.Targets{common.TargetCheck}},
				},
				{
					name:   "renders the hooks of the stage pre-commit alone",
					golden: "pre-commit.yaml",
					give:   common.Hooks{PreCommit: common.Targets{common.TargetLint}},
				},
			}
			for _, tt := range hooks {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					hooked := adjusted{adjust: func(o *common.Options) { o.Hooks = tt.give }}
					p := baseline.Producer{Name: common.Name, Producer: hooked}
					dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), p)
					baselinetest.Hygiene(t, dir)
					got := files.Read(t, filepath.Join(dir, preCommitConfig))
					golden.MatchAt(t, filepath.Join("testdata", "golden", "hooks", tt.golden), []byte(got),
						golden.ShouldUpdate())
				})
			}

			t.Run("runs ergon with the command of the options in the Makefile and the hooks", func(t *testing.T) {
				t.Parallel()
				source := adjusted{adjust: func(o *common.Options) {
					o.Ergon = common.Command{"go", "run", "go.dokimi.dev/ergon/cmd/ergon"}
				}}
				p := baseline.Producer{Name: common.Name, Producer: source}
				dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), p)
				baselinetest.Hygiene(t, dir)
				assert.Contains(t, files.Read(t, filepath.Join(dir, makefile)),
					"\nERGON ?= go run go.dokimi.dev/ergon/cmd/ergon\n", "the command of ergon in the Makefile")
				assert.Contains(
					t,
					files.Read(t, filepath.Join(dir, preCommitConfig)),
					"\n        entry: go run go.dokimi.dev/ergon/cmd/ergon tool run common.commitlint -- lint --message\n",
					"the command of the commit-msg hook",
				)
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the hooks lint and test before a commit and check before a push", func(t *testing.T) {
				t.Parallel()
				o, _ := common.Producer{}.Options().(*common.Options)
				assert.Equal(t, o.Hooks, common.Hooks{
					PreCommit: common.Targets{common.TargetLint, common.TargetTest},
					PrePush:   common.Targets{common.TargetCheck},
				}, "the hooks")
			})

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := common.Producer{}.Options().(*common.Options)
				second, _ := common.Producer{}.Options().(*common.Options)
				first.Tools.Commitlint.SHA256["linux/amd64"] = "changed"
				assert.NotEqual(
					t,
					second.Tools.Commitlint.SHA256["linux/amd64"],
					"changed",
					"the digest of the second value",
				)
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the jobs docs, commits and changeset as checks of text", func(t *testing.T) {
				t.Parallel()
				got := common.Producer{}.Contribution(common.Producer{}.Options())
				assert.Equal(t, got, workflow.Contribution{Jobs: []workflow.Job{
					{
						ID:          "docs",
						Name:        "Docs",
						Text:        true,
						Timeout:     10,
						Permissions: read,
						Steps: []workflow.Step{
							{Uses: markdownlint, With: map[string]string{"globs": "**/*.md\n#node_modules"}},
						},
					},
					{
						ID:          "commits",
						Name:        "Commits",
						If:          "github.event_name == 'pull_request' && github.event.pull_request.user.login != 'dependabot[bot]'",
						Text:        true,
						Timeout:     10,
						Permissions: read,
						History:     true,
						Ergon:       true,
						Tools:       true,
						Steps: []workflow.Step{{
							Name: "Check the commit messages",
							Env: map[string]string{
								"BASE": "${{ github.event.pull_request.base.sha }}",
								"HEAD": "${{ github.event.pull_request.head.sha }}",
							},
							Run: []string{
								"status=0",
								`for commit in $(git rev-list --no-merges "$BASE..$HEAD"); do`,
								`  git log -1 --format=%B "$commit" | ergon tool run common.commitlint -- lint || status=1`,
								"done",
								`exit "$status"`,
							},
						}},
					},
					{
						ID:   "changeset",
						Name: "Changeset",
						If: "github.event_name == 'pull_request' && " +
							"github.event.pull_request.user.login != 'dependabot[bot]' && " +
							"!(github.event.pull_request.head.repo.full_name == github.repository && " +
							"github.head_ref == format('ergon-release/{0}', github.base_ref))",
						Text:        true,
						Timeout:     10,
						Permissions: read,
						History:     true,
						Ergon:       true,
						Steps: []workflow.Step{{
							Name: "Check the changesets",
							Env:  map[string]string{"BASE": "${{ github.event.pull_request.base.sha }}"},
							Run:  []string{`ergon release status --since "$BASE"`},
						}},
					},
				}}, "the contribution")
			})

			t.Run("returns a valid contribution", func(t *testing.T) {
				t.Parallel()
				got := common.Producer{}.Contribution(common.Producer{}.Options())
				assert.NoError(t, got.Validate(), "Validate of the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				want := common.Producer{}.Contribution(common.Producer{}.Options())
				assert.Equal(t, common.Producer{}.Contribution(nil), want, "the contribution for nil options")
			})

			t.Run("returns the jobs with the timeout of the options", func(t *testing.T) {
				t.Parallel()
				o, _ := common.Producer{}.Options().(*common.Options)
				o.CI.Timeout = 25
				got := common.Producer{}.Contribution(o)
				assert.Equal(t, got.Jobs[0].Timeout, 25, "the timeout of docs")
				assert.Equal(t, got.Jobs[1].Timeout, 25, "the timeout of commits")
				assert.Equal(t, got.Jobs[2].Timeout, 25, "the timeout of changeset")
			})
		})
	})
}

// producer returns the producer of the common files as a base producer.
func producer() baseline.Producer {
	return baseline.Producer{Name: common.Name, Producer: common.Producer{}}
}
