// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/common"
)

// name pins the name of the producer of the common files.
const name = "common"

// markdownlint is the pin of the action of markdownlint at the baseline.
var markdownlint = workflow.Action{
	Uses:    "DavidAnson/markdownlint-cli2-action",
	Commit:  "21c1be1b93ad9ed58fa840aacc3f279cde2a72ff",
	Release: "v24.2.0",
}

// read is the permission of the jobs of the common files.
var read = map[string]string{"contents": "read"}

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
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

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
