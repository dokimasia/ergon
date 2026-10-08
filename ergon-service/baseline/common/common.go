// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	"embed"
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/release"
)

// Name is the name of the producer of the common files in the lock, and of its section of
// .ergon.yaml.
const Name = "common"

// templates are the templates of the common files, under managed/, seeded/ and shared/.
//
//go:embed all:templates
var templates embed.FS

// read is the permission of a job to read the contents of the repository.
var read = map[string]string{"contents": "read"}

// Producer renders the common files of a repository: the files that do not depend on its languages
// or its forge. Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of the common files: the managed files under managed/, the
// seeded files under seeded/, and the first fragments of the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the options of the common files at the baseline: the release of commitlint with
// the digests that it states, the releases of pre-commit-hooks and markdownlint-cli2-action, and a
// limit of 10 minutes for each job.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{Commitlint: Commitlint{Binary: option.Binary{
			SHA256: map[option.Platform]string{
				option.LinuxAMD64:   "bf9666441262bf8d31345a6d29d6f788d0dadea7d47e5b91890c3a651999c118",
				option.DarwinARM64:  "35f25a55947031db016177ac0211d479c1fca21ccb87b368b6a429629524436c",
				option.WindowsAMD64: "e8f90d08616ab8cd987b2dbef7ae0ee7fd1cb43c97f5a1ad171e8a56d1bd2f3d",
			},
			Version: "0.12.0",
		}}},
		PreCommitHooks: "v6.0.0",
		CI: option.CI[Actions]{
			Actions: Actions{Markdownlint: workflow.Action{
				Uses:    "DavidAnson/markdownlint-cli2-action",
				Commit:  "21c1be1b93ad9ed58fa840aacc3f279cde2a72ff",
				Release: "v24.2.0",
			}},
			Timeout: 10,
		},
	}
}

// Contribution returns the jobs of the common files in ci.yml for o, and for the options at the
// baseline when o is not the options of the common files. The jobs check text, so they run on the
// Linux runner of the section github:
//
//   - docs lints the Markdown files with the markdownlint of o.
//   - commits checks each commit message of a pull request with the commitlint of o, through ergon
//     tool run, and keeps commitlint in the cache of GitHub Actions. It skips the pull requests of
//     Dependabot, whose bodies exceed the length of a line.
//   - changeset runs ergon release status against the base of a pull request, which fails for a
//     package that the pull request changes without a changeset that names it. It skips the pull
//     requests of Dependabot, because the next release of each module includes their updates.
//     It also skips the version pull request that ergon release ci version opens from the branch
//     ergon-release/<base> of the repository, which removes the changesets that it releases.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	docs := workflow.Job{
		ID:          "docs",
		Name:        "Docs",
		Text:        true,
		Timeout:     opts.CI.Timeout,
		Permissions: read,
		Steps: []workflow.Step{{
			Uses: opts.CI.Actions.Markdownlint,
			With: map[string]string{"globs": "**/*.md\n#node_modules"},
		}},
	}
	commits := workflow.Job{
		ID:          "commits",
		Name:        "Commits",
		If:          "github.event_name == 'pull_request' && github.event.pull_request.user.login != 'dependabot[bot]'",
		Text:        true,
		Timeout:     opts.CI.Timeout,
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
				`  git log -1 --format=%B "$commit" | ergon tool run ` + Name + `.commitlint -- lint || status=1`,
				"done",
				`exit "$status"`,
			},
		}},
	}
	changeset := workflow.Job{
		ID:   "changeset",
		Name: "Changeset",
		If: "github.event_name == 'pull_request' && github.event.pull_request.user.login != 'dependabot[bot]' && " +
			"!(github.event.pull_request.head.repo.full_name == github.repository && " +
			"github.head_ref == format('" + release.ReleaseBranch + "{0}', github.base_ref))",
		Text:        true,
		Timeout:     opts.CI.Timeout,
		Permissions: read,
		History:     true,
		Ergon:       true,
		Steps: []workflow.Step{{
			Name: "Check the changesets",
			Env:  map[string]string{"BASE": "${{ github.event.pull_request.base.sha }}"},
			Run:  []string{`ergon release status --since "$BASE"`},
		}},
	}
	return workflow.Contribution{Jobs: []workflow.Job{docs, commits, changeset}}
}
