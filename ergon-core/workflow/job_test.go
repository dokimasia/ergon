// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

// setupGo is the action of the setup of Go in the cases.
var setupGo = workflow.Action{
	Uses:    "actions/setup-go",
	Commit:  "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
	Release: "v7.0.0",
}

func TestJob(t *testing.T) {
	t.Parallel()

	t.Run("Setup", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give func(*workflow.Setup)
			}{
				{
					name: "returns nil for a setup with version files, runners, versions, variables and steps",
					give: func(*workflow.Setup) {},
				},
				{
					name: "returns nil for a setup of a timeout alone",
					give: func(s *workflow.Setup) { *s = workflow.Setup{Timeout: 1} },
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					s := goSetup()
					tt.give(s)
					assert.NoError(t, s.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give func(*workflow.Setup)
			}{
				{
					name: "returns ErrInvalidJob for files with a single quote",
					give: func(s *workflow.Setup) { s.Files = "a'b" },
				},
				{
					name: "returns ErrInvalidJob for files that span lines",
					give: func(s *workflow.Setup) { s.Files = "a\nb" },
				},
				{
					name: "returns ErrInvalidJob for version files with a single quote",
					give: func(s *workflow.Setup) { s.VersionFiles = "go'work" },
				},
				{
					name: "returns ErrInvalidJob for version files that span lines",
					give: func(s *workflow.Setup) { s.VersionFiles = "go.work\ngo.mod" },
				},
				{
					name: "returns ErrInvalidJob for a runner with a space",
					give: func(s *workflow.Setup) { s.Runners = []string{"ubuntu 26.04"} },
				},
				{
					name: "returns ErrInvalidJob for a runner named twice",
					give: func(s *workflow.Setup) { s.Runners = []string{"macos-26", "macos-26"} },
				},
				{
					name: "returns ErrInvalidJob for an empty version",
					give: func(s *workflow.Setup) { s.Versions = []string{""} },
				},
				{
					name: "returns ErrInvalidJob for a version that spans lines",
					give: func(s *workflow.Setup) { s.Versions = []string{"1.27\n"} },
				},
				{
					name: "returns ErrInvalidJob for a version named twice",
					give: func(s *workflow.Setup) { s.Versions = []string{"1.27", "1.27"} },
				},
				{name: "returns ErrInvalidJob for a timeout of 0", give: func(s *workflow.Setup) { s.Timeout = 0 }},
				{
					name: "returns ErrInvalidJob for a variable whose name has a hyphen",
					give: func(s *workflow.Setup) { s.Env = map[string]string{"GO-FLAGS": "-mod=mod"} },
				},
				{
					name: "returns ErrInvalidJob for a step that is not valid",
					give: func(s *workflow.Setup) { s.Steps = []workflow.Step{{Name: "nothing"}} },
				},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					s := goSetup()
					tt.give(s)
					assert.ErrorIs(t, s.Validate(), workflow.ErrInvalidJob, "Validate")
				})
			}

			t.Run("returns ErrInvalidStep for a step that is not valid", func(t *testing.T) {
				t.Parallel()
				s := goSetup()
				s.Steps = []workflow.Step{{Name: "nothing"}}
				assert.ErrorIs(t, s.Validate(), workflow.ErrInvalidStep, "Validate")
			})
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give func(*workflow.Job)
		}{
			{name: "returns nil for a job with its own setup", give: func(*workflow.Job) {}},
			{
				name: "returns nil for a job that names a toolchain",
				give: func(j *workflow.Job) { j.Setup, j.Toolchain = nil, "jvm" },
			},
			{
				name: "returns nil for a check of text with a timeout",
				give: func(j *workflow.Job) { j.Setup, j.Text, j.Timeout = nil, true, 10 },
			},
			{
				name: "returns nil for a job on every runner with a timeout",
				give: func(j *workflow.Job) { j.Setup, j.Timeout, j.Ergon = nil, 15, true },
			},
			{
				name: "returns nil for a job with a setup that runs tools",
				give: func(j *workflow.Job) { j.Tools = true },
			},
			{
				name: "returns nil for a check of text that runs ergon and its tools",
				give: func(j *workflow.Job) { j.Setup, j.Text, j.Timeout, j.Ergon, j.Tools = nil, true, 10, true, true },
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				j := goJob()
				tt.give(j)
				assert.NoError(t, j.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give func(*workflow.Job)
		}{
			{
				name: "returns ErrInvalidJob for an id that starts with a digit",
				give: func(j *workflow.Job) { j.ID = "1check" },
			},
			{name: "returns ErrInvalidJob for an empty name", give: func(j *workflow.Job) { j.Name = "" }},
			{
				name: "returns ErrInvalidJob for a name that spans lines",
				give: func(j *workflow.Job) { j.Name = "Go\n" },
			},
			{
				name: "returns ErrInvalidJob for a condition that spans lines",
				give: func(j *workflow.Job) { j.If = "a\nb" },
			},
			{
				name: "returns ErrInvalidJob for a toolchain with an uppercase letter",
				give: func(j *workflow.Job) { j.Setup, j.Toolchain = nil, "JVM" },
			},
			{
				name: "returns ErrInvalidJob for a setup and a toolchain",
				give: func(j *workflow.Job) { j.Toolchain = "go" },
			},
			{
				name: "returns ErrInvalidJob for a setup and a check of text",
				give: func(j *workflow.Job) { j.Text = true },
			},
			{
				name: "returns ErrInvalidJob for a toolchain and a check of text",
				give: func(j *workflow.Job) { j.Setup, j.Toolchain, j.Text = nil, "jvm", true },
			},
			{
				name: "returns ErrInvalidJob for a timeout of a job with a setup",
				give: func(j *workflow.Job) { j.Timeout = 30 },
			},
			{
				name: "returns ErrInvalidJob for no timeout of a job without a setup",
				give: func(j *workflow.Job) { j.Setup, j.Text = nil, true },
			},
			{
				name: "returns ErrInvalidJob for tools of a job without a setup that does not run ergon",
				give: func(j *workflow.Job) { j.Setup, j.Text, j.Timeout, j.Tools = nil, true, 10, true },
			},
			{
				name: "returns ErrInvalidJob for a setup that is not valid",
				give: func(j *workflow.Job) { j.Setup.Timeout = 0 },
			},
			{
				name: "returns ErrInvalidJob for a scope in uppercase",
				give: func(j *workflow.Job) { j.Permissions = map[string]string{"Contents": "read"} },
			},
			{
				name: "returns ErrInvalidJob for an access other than read, write and none",
				give: func(j *workflow.Job) { j.Permissions = map[string]string{"contents": "admin"} },
			},
			{name: "returns ErrInvalidJob for no step", give: func(j *workflow.Job) { j.Steps = nil }},
			{
				name: "returns ErrInvalidJob for a step that is not valid",
				give: func(j *workflow.Job) { j.Steps = []workflow.Step{{Run: []string{""}}} },
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				j := goJob()
				tt.give(j)
				assert.ErrorIs(t, j.Validate(), workflow.ErrInvalidJob, "Validate")
			})
		}
	})
}

// goSetup returns a new valid setup of the cases: Go on two runners and two versions, with an
// environment variable and the step that installs Go.
func goSetup() *workflow.Setup {
	return &workflow.Setup{
		Files:        "**/go.mod",
		VersionFiles: "go.work",
		Runners:      []string{"ubuntu-26.04", "macos-26"},
		Versions:     []string{"1.26", "1.27"},
		Timeout:      30,
		Env:          map[string]string{"GOTOOLCHAIN": "local"},
		Steps:        []workflow.Step{{Uses: setupGo, With: map[string]string{"go-version": "${{ matrix.version }}"}}},
	}
}

// goJob returns a new valid job of the cases: the check of Go with its own setup, which runs make.
func goJob() *workflow.Job {
	return &workflow.Job{
		ID:          "check-go",
		Name:        "Go",
		Setup:       goSetup(),
		Permissions: map[string]string{"contents": "read"},
		Steps:       []workflow.Step{{Run: []string{"make check-go"}}},
	}
}
