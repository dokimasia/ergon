// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github_test

import (
	"regexp"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.yaml.in/yaml/v3"
)

// managed is the comment that opens a managed file.
const managed = "Managed by ergon init. Add repository settings to .ergon/local/"

// The rules of every workflow, as the cases check them.
const (
	// group is the concurrency group of a workflow that an event triggers: the workflow and the ref.
	group = "${{ github.workflow }}-${{ github.ref }}"

	// cancel cancels a superseded run on a pull request only.
	cancel = "${{ github.event_name == 'pull_request' }}"
)

// workflows are the paths of the workflows of the GitHub files.
var workflows = []string{
	".github/workflows/baseline.yml",
	language.CI,
	".github/workflows/codeql.yml",
	language.Security,
}

// uses matches a line that runs an action, and captures the reference of the action with its
// comment.
var uses = regexp.MustCompile(`(?m)^\s*(?:- )?uses: (.+)$`)

// pinned matches the reference of an action pinned to the commit of a release, with the release
// in a comment, and a reference to an action or a workflow of the repository.
var pinned = regexp.MustCompile(`^(?:[A-Za-z0-9-]+/[A-Za-z0-9._/-]+@[0-9a-f]{40} # v\d+\.\d+\.\d+|\./\.github/\S+)$`)

// workflow is the part of a workflow that the cases check.
type workflow struct {
	// On are the events that trigger the workflow.
	On map[string]any `yaml:"on"`

	// Permissions are the permissions at the top level of the workflow.
	Permissions map[string]string `yaml:"permissions"`

	// Concurrency is the concurrency of the workflow.
	Concurrency struct {
		// Group is the concurrency group.
		Group string `yaml:"group"`

		// CancelInProgress is the expression that cancels a superseded run.
		CancelInProgress string `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`

	// Jobs are the jobs of the workflow, by their identifier.
	Jobs map[string]struct {
		// RunsOn is the runner of the job.
		RunsOn string `yaml:"runs-on"`

		// Strategy is the strategy of the job, with the systems of its matrix.
		Strategy struct {
			// Matrix is the matrix of the job.
			Matrix struct {
				// OS are the runners of the matrix.
				OS []string `yaml:"os"`
			} `yaml:"matrix"`
		} `yaml:"strategy"`

		// TimeoutMinutes is the timeout of the job.
		TimeoutMinutes int `yaml:"timeout-minutes"`

		// Permissions are the permissions of the job.
		Permissions map[string]string `yaml:"permissions"`

		// Steps are the steps of the job.
		Steps []struct {
			// Uses is the action of the step.
			Uses string `yaml:"uses"`

			// With are the inputs of the action.
			With map[string]any `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestGitHub(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("Files", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the managed and seeded GitHub files", func(t *testing.T) {
				t.Parallel()
				classes := map[string]language.Class{}
				for path, f := range render(t) {
					classes[path] = f.Class
				}
				assert.Equal(t, classes, map[string]language.Class{
					".github/ISSUE_TEMPLATE/bug.yml":         language.Managed,
					".github/ISSUE_TEMPLATE/config.yml":      language.Managed,
					".github/ISSUE_TEMPLATE/feature.yml":     language.Managed,
					".github/PULL_REQUEST_TEMPLATE.md":       language.Managed,
					".github/actions/setup-ergon/action.yml": language.Managed,
					".github/actions/setup-make/action.yml":  language.Managed,
					".github/dependabot.yml":                 language.Managed,
					".github/workflows/baseline.yml":         language.Managed,
					".github/workflows/ci.yml":               language.Managed,
					".github/workflows/codeql.yml":           language.Managed,
					".github/workflows/security.yml":         language.Managed,
					".github/CODEOWNERS":                     language.Seeded,
				}, "the classes of the GitHub files")
			})

			t.Run("renders the shared files as fragments", func(t *testing.T) {
				t.Parallel()
				files := render(t)
				for _, path := range []string{language.CI, language.Security, language.Dependabot} {
					assert.True(t, files[path].Content == nil && files[path].Fragment != nil, "the fragment of "+path)
				}
				assert.True(t, files[".github/CODEOWNERS"].Fragment == nil, "the fragment of CODEOWNERS")
			})

			t.Run("opens each managed file with the managed comment", func(t *testing.T) {
				t.Parallel()
				for path, f := range render(t) {
					if f.Class != language.Managed {
						continue
					}
					first, _, _ := strings.Cut(text(f), "\n")
					assert.Contains(t, first, managed+path+" and run ergon init sync.", "the first line of "+path)
				}
			})

			t.Run("fills the repository into the link of a vulnerability report", func(t *testing.T) {
				t.Parallel()
				config := text(render(t)[".github/ISSUE_TEMPLATE/config.yml"])
				assert.Contains(t, config, "url: https://github.com/dokimasia/demo/security/advisories/new\n",
					"the config.yml of the issue forms")
			})

			t.Run("leaves no placeholder in a file", func(t *testing.T) {
				t.Parallel()
				for path, f := range render(t) {
					assert.NotContains(t, strings.ReplaceAll(text(f), "${{", ""), "{{", "the placeholders of "+path)
				}
			})

			t.Run("renders each YAML file as one document", func(t *testing.T) {
				t.Parallel()
				for path, f := range render(t) {
					if !strings.HasSuffix(path, ".yml") {
						continue
					}
					var document map[string]any
					assert.NoError(t, yaml.Unmarshal([]byte(text(f)), &document), "Unmarshal of "+path)
					assert.NotEmpty(t, document, "the document of "+path)
				}
			})

			t.Run("pins every action to the commit of a release", func(t *testing.T) {
				t.Parallel()
				var references []string
				for _, f := range render(t) {
					for _, match := range uses.FindAllStringSubmatch(text(f), -1) {
						references = append(references, match[1])
					}
				}
				assert.Contains(t, references, language.Checkout, "the references of the actions")
				for _, reference := range references {
					assert.True(t, pinned.MatchString(reference), "the pin of "+reference)
				}
			})

			t.Run("grants no permission at the top level of a workflow", func(t *testing.T) {
				t.Parallel()
				for path, w := range parse(t) {
					assert.True(t, w.Permissions != nil && len(w.Permissions) == 0, "the permissions of "+path)
				}
			})

			t.Run("gives every job a timeout and its own permissions", func(t *testing.T) {
				t.Parallel()
				for path, w := range parse(t) {
					for id, job := range w.Jobs {
						assert.True(t, job.TimeoutMinutes > 0, "the timeout of "+path+" "+id)
						assert.NotEmpty(t, job.Permissions, "the permissions of "+path+" "+id)
					}
				}
			})

			t.Run("runs the baseline on Linux, macOS and Windows", func(t *testing.T) {
				t.Parallel()
				job := parse(t)[language.CI].Jobs["baseline"]
				assert.Equal(t, job.RunsOn, "${{ matrix.os }}", "the runner of the baseline")
				assert.Equal(t, job.Strategy.Matrix.OS, []string{language.Linux, language.MacOS, language.Windows},
					"the systems of the baseline")
			})

			t.Run("runs every other job on Linux", func(t *testing.T) {
				t.Parallel()
				for path, w := range parse(t) {
					for id, job := range w.Jobs {
						if path == language.CI && id == "baseline" {
							continue
						}
						assert.Equal(t, job.RunsOn, language.Linux, "the runner of "+path+" "+id)
					}
				}
			})

			t.Run("checks out without persisting the credentials", func(t *testing.T) {
				t.Parallel()
				for path, w := range parse(t) {
					for id, job := range w.Jobs {
						assert.True(t, strings.HasPrefix(job.Steps[0].Uses, "actions/checkout@"),
							"the first step of "+path+" "+id)
						assert.Equal(t, job.Steps[0].With["persist-credentials"], any(false),
							"persist-credentials of "+path+" "+id)
					}
				}
			})

			t.Run("groups the runs of a workflow that an event triggers by workflow and ref", func(t *testing.T) {
				t.Parallel()
				for path, w := range parse(t) {
					if _, called := w.On["workflow_call"]; called {
						continue
					}
					assert.Equal(t, w.Concurrency.Group, group, "the concurrency group of "+path)
					assert.Equal(t, w.Concurrency.CancelInProgress, cancel, "cancel-in-progress of "+path)
					assert.False(t, w.On["pull_request_target"] != nil, "pull_request_target in "+path)
				}
			})

			t.Run("returns ErrInvalidAnswer for a repository that is not owner/name", func(t *testing.T) {
				t.Parallel()
				a := answers()
				a.Repository = "demo"
				_, err := github.Initializer{}.Files(a)
				assert.ErrorIs(t, err, github.ErrInvalidAnswer, "Files")
			})
		})
	})
}

// answers returns new answers of the cases, which a case may change.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Languages:       []workspace.Language{"go"},
		Owner:           "Dokimasia B.V.",
		License:         "MIT",
		Year:            2026,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
	}
}

// render returns the GitHub files for the answers of the cases, by path.
func render(t *testing.T) map[string]language.File {
	t.Helper()
	files, err := github.Initializer{}.Files(answers())
	assert.NoError(t, err, "Files")
	byPath := make(map[string]language.File, len(files))
	for _, f := range files {
		byPath[f.Path] = f
	}
	return byPath
}

// parse returns the workflows of the GitHub files, by path.
func parse(t *testing.T) map[string]workflow {
	t.Helper()
	files := render(t)
	parsed := make(map[string]workflow, len(workflows))
	for _, path := range workflows {
		var w workflow
		assert.NoError(t, yaml.Unmarshal([]byte(text(files[path])), &w), "Unmarshal of "+path)
		parsed[path] = w
	}
	return parsed
}

// text returns the content or the fragment of f.
func text(f language.File) string {
	return string(f.Content) + string(f.Fragment)
}
