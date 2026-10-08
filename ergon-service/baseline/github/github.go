// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github

import (
	"embed"
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// Name is the name of the producer of the GitHub files in the lock, and of its section of
// .ergon.yaml.
const Name = "github"

// templates are the templates of the GitHub files, under managed/ and seeded/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the GitHub files of a repository: the workflows, the actions that install ergon
// and GNU make, the configuration of Dependabot, the issue forms, the template of a pull request,
// and CODEOWNERS. Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Calculator   = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of the GitHub files: the managed files under managed/.github/,
// and CODEOWNERS under seeded/.github/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the options of the GitHub files at the baseline: the runners ubuntu-26.04,
// macos-26 and windows-2025, ubuntu-26.04 for the checks of text, GNU make 4.4.1, the pins of
// actions/checkout v7.0.1, github/codeql-action v4.38.2, actions/dependency-review-action v5.0.0,
// ossf/scorecard-action v2.4.4, actions/upload-artifact v7.0.2 and actions/download-artifact
// v8.0.2, and a limit of 15 minutes for each job.
func (Producer) Options() language.Options {
	return &Options{
		Runners: option.Runners{"ubuntu-26.04", "macos-26", "windows-2025"},
		Linux:   "ubuntu-26.04",
		Make:    "4.4.1",
		CI: option.CI[Actions]{
			Actions: Actions{
				Checkout: workflow.Action{
					Uses:    "actions/checkout",
					Commit:  "3d3c42e5aac5ba805825da76410c181273ba90b1",
					Release: "v7.0.1",
				},
				CodeQL: workflow.Action{
					Uses:    "github/codeql-action",
					Commit:  "2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2",
					Release: "v4.38.2",
				},
				DependencyReview: workflow.Action{
					Uses:    "actions/dependency-review-action",
					Commit:  "a1d282b36b6f3519aa1f3fc636f609c47dddb294",
					Release: "v5.0.0",
				},
				Scorecard: workflow.Action{
					Uses:    "ossf/scorecard-action",
					Commit:  "2d1146689b8cda280b9bc96326124645441f03bc",
					Release: "v2.4.4",
				},
				UploadArtifact: workflow.Action{
					Uses:    "actions/upload-artifact",
					Commit:  "cf430e030ddbb5b0abf93d22962f4752f3646cd9",
					Release: "v7.0.2",
				},
				DownloadArtifact: workflow.Action{
					Uses:    "actions/download-artifact",
					Commit:  "9000827ccba6bdab643e8b6fd33ac0654aef8333",
					Release: "v8.0.2",
				},
			},
			Timeout: 15,
		},
	}
}

// Data returns the jobs of ci.yml, which its template reads as .Data: the jobs of c, as
// [Options.Jobs] returns them for o, and for the options at the baseline when o is not the options
// of the GitHub files. It returns the error of Options.Jobs for a job whose runners the section
// does not list.
func (Producer) Data(_ *language.Answers, o language.Options, c *workflow.Contribution) (any, error) {
	return own(o).Jobs(c)
}

// Contribution returns the job baseline of ci.yml for o, and for the options at the baseline when o
// is not the options of the GitHub files. The job runs ergon init check on the Linux runner of the
// section, as a check of text: the managed .gitattributes checks out every text file with LF on
// each system, so the check compares the same bytes everywhere.
func (Producer) Contribution(o language.Options) workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:          "baseline",
		Name:        "Baseline",
		Text:        true,
		Timeout:     own(o).CI.Timeout,
		Permissions: map[string]string{"contents": "read"},
		Ergon:       true,
		Steps:       []workflow.Step{{Name: "Check the managed files", Run: []string{"ergon init check"}}},
	}}}
}

// own returns o as the options of the GitHub files, and the options at the baseline when o is not.
func own(o language.Options) *Options {
	if opts, ok := o.(*Options); ok {
		return opts
	}
	opts, _ := Producer{}.Options().(*Options)
	return opts
}
