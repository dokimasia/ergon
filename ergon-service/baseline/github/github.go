// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.yaml.in/yaml/v3"
)

// Name is the name of the producer of the GitHub files in the lock, and of its section of
// .ergon.yaml.
const Name = "github"

// dependabot is the path of the configuration of Dependabot.
const dependabot = ".github/dependabot.yml"

// The condition of each step of a CodeQL analysis in codeql.yml, before and after the language of
// the analysis: the run analyzes that language, and the repository has a file of the pattern of the
// files of the analysis.
const (
	codeqlLanguage = "inputs.language == '"
	codeqlFiles    = "' && hashFiles(inputs.files) != ''"
)

// rendered are the files that ergon init renders of each ecosystem of Dependabot that edits them.
var rendered = map[string]string{
	"github-actions": "the workflows and the actions under .github/",
	"pre-commit":     ".pre-commit-config.yaml",
}

// templates are the templates of the GitHub files, under managed/ and seeded/.
//
//go:embed all:templates
var templates embed.FS

// Workflows are the data of the templates of the GitHub files, which each template reads as .Data.
type Workflows struct {
	// TapOwner and TapName are the owner and the name of the Homebrew tap of the casks of the
	// assets, whose job homebrew release.yml has, or empty.
	TapOwner, TapName string

	// Jobs are the jobs of ci.yml, as [Options.Jobs] returns them.
	Jobs []Job

	// Nightly are the jobs of nightly.yml, as [Options.NightlyJobs] returns them. ergon init renders
	// no nightly.yml without one.
	Nightly []Job

	// CodeQL are the steps that codeql.yml runs before an analysis: the steps of each CodeQL
	// analysis of the contributions, in their order. Each step runs only when the run analyzes the
	// language of its analysis and the repository has a file of the files of its analysis.
	CodeQL []workflow.Step

	// Assets reports that a producer builds release assets in the job pack of release.yml, which
	// then attests them.
	Assets bool
}

// Producer renders the GitHub files of a repository: the workflows, the actions that install ergon
// and GNU make, the configuration of Dependabot, the issue forms, the template of a pull request,
// and CODEOWNERS. Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Calculator   = Producer{}
	_ language.Contributor  = Producer{}
	_ language.LocalChecker = Producer{}
)

// Templates returns the templates of the GitHub files: the managed files under managed/.github/,
// and CODEOWNERS under seeded/.github/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the options of the GitHub files at the baseline: the runners ubuntu-26.04,
// macos-26 and windows-2025, ubuntu-26.04 for the checks of text, the release of GNU make, the
// releases of actions/checkout, github/codeql-action, actions/dependency-review-action,
// ossf/scorecard-action, actions/upload-artifact, actions/download-artifact, actions/cache,
// actions/create-github-app-token and actions/attest, a limit of 15 minutes for each job, and
// nightly.yml at 03:00 UTC each day.
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
				Cache: workflow.Action{
					Uses:    "actions/cache",
					Commit:  "55cc8345863c7cc4c66a329aec7e433d2d1c52a9",
					Release: "v6.1.0",
				},
				CreateGitHubAppToken: workflow.Action{
					Uses:    "actions/create-github-app-token",
					Commit:  "bcd2ba49218906704ab6c1aa796996da409d3eb1",
					Release: "v3.2.0",
				},
				Attest: workflow.Action{
					Uses:    "actions/attest",
					Commit:  "1e69f48acb82d1966a394da916b4c1698aa569d6",
					Release: "v4.2.2",
				},
			},
			Timeout: 15,
		},
		Nightly: Nightly{Schedule: "0 3 * * *"},
	}
}

// Data returns the [Workflows] of c for o, and for the options at the baseline when o is not the
// options of the GitHub files: the jobs of c, as [Options.Jobs] returns them, its nightly jobs, as
// [Options.NightlyJobs] returns them, the steps of its CodeQL analyses, and its assets with the
// owner and the name of their tap. Each step of an analysis is a copy, with the condition of its
// language and its files before its own condition. Data returns the error of Jobs or NightlyJobs
// for a job whose runners the section does not list.
func (Producer) Data(_ *language.Answers, o language.Options, c *workflow.Contribution) (any, error) {
	jobs, err := own(o).Jobs(c)
	if err != nil {
		return nil, err
	}
	nightly, err := own(o).NightlyJobs(c)
	if err != nil {
		return nil, err
	}
	w := Workflows{Jobs: jobs, Nightly: nightly, Assets: c.Assets != nil}
	for i := range c.CodeQL {
		a := &c.CodeQL[i]
		w.CodeQL = append(w.CodeQL, guard(a.Steps, codeqlLanguage+a.Language+codeqlFiles)...)
	}
	if c.Assets != nil {
		w.TapOwner, w.TapName, _ = strings.Cut(c.Assets.Tap, "/")
	}
	return w, nil
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

// CheckLocal returns an error that wraps [language.ErrInvalidLocal] for content, the configuration
// of Dependabot at path with its local file merged in, that has an update of the ecosystem
// github-actions or pre-commit. ergon init renders the files of both ecosystems, so a pull request of
// Dependabot would edit a managed file. The error names the ecosystem and the files. CheckLocal also
// returns one for updates that do not decode, and nil for every other path.
func (Producer) CheckLocal(path string, content []byte) error {
	if path != dependabot {
		return nil
	}
	var config struct {
		// Updates are the updates of the configuration.
		Updates []struct {
			// Ecosystem is the package manager of the update, such as gomod.
			Ecosystem string `yaml:"package-ecosystem"`
		} `yaml:"updates"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("%w: the updates of %s: %w", language.ErrInvalidLocal, path, err)
	}
	for _, u := range config.Updates {
		if files, ok := rendered[u.Ecosystem]; ok {
			return fmt.Errorf("%w: %s updates the ecosystem %s, so Dependabot would edit %s, which ergon init "+
				"renders", language.ErrInvalidLocal, path, u.Ecosystem, files)
		}
	}
	return nil
}

// own returns o as the options of the GitHub files, and the options at the baseline when o is not.
func own(o language.Options) *Options {
	if opts, ok := o.(*Options); ok {
		return opts
	}
	opts, _ := Producer{}.Options().(*Options)
	return opts
}
