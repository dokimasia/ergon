// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"embed"
	"io/fs"
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// Name is the name of the producer of spec in the lock, and of its section of .ergon.yaml.
const Name = "spec"

// templates are the templates of spec, under managed/ and shared/.
//
//go:embed all:templates
var templates embed.FS

// read is the permission of the job spec to read the contents of the repository.
var read = map[string]string{"contents": "read"}

// Repository are the data of the templates of spec: what the repository is to spec, which
// [Producer.Data] computes from the roles of the options.
type Repository struct {
	// Implementation reports that the repository has copies of definitions.
	Implementation bool

	// Definition reports that the repository is the repository of a definition.
	Definition bool
}

// Producer renders the files of spec in a repository with a role: the targets of spec in the
// Makefile, the job spec of ci.yml, and spec.yml in an implementation. Its zero value is ready to
// use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Calculator   = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of spec: spec.yml under managed/.github/workflows/, and the
// fragment of the Makefile with the targets of spec under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the options of spec at the baseline: no role, the release 0.2.0 of spec with the
// digests that it states for every platform, the pin of propose.yml of that release, and a limit of
// 10 minutes for the job spec.
func (Producer) Options() language.Options {
	return &Options{
		Roles: []Role{},
		Tools: Tools{Spec: Spec{Binary: option.Binary{
			SHA256: map[option.Platform]string{
				option.LinuxAMD64:   "256471ec74a0c622959072ca63f8ed4c663ff8a2182b4f367e8c344df2b54b1c",
				option.LinuxARM64:   "ea90695c408e9766fdc9efd83cd7902279289fa055f37d30b1b5ee39244bd9ce",
				option.DarwinAMD64:  "15732d32ba1be34c1ed54160528b07ffb07f20edcc1f021cb8eb6f6a8cc56059",
				option.DarwinARM64:  "47b21e000521e59b53d339960aff3acb3b37df3cd8245a4a0de66a682f07b8b6",
				option.WindowsAMD64: "17037298e47775de63981b393c5c52f09f091aed7be560ceb0877b827745afc0",
				option.WindowsARM64: "d62da3e454ae9ebb928bc3c7860a6035af9a00569ea7b4a42106cee133145452",
			},
			Version: "0.2.0",
		}}},
		CI: option.CI[Actions]{
			Actions: Actions{Propose: workflow.Action{
				Uses:    "stealth-scale/spec/.github/workflows/propose.yml",
				Commit:  "78d70fd5be63102e43f8dd9628d4594914bb390e",
				Release: "v0.2.0",
			}},
			Timeout: 10,
		},
	}
}

// Data returns the [Repository] of the roles of o, and of the options at the baseline when o is not
// the options of spec. It does not fail.
func (Producer) Data(_ *language.Answers, o language.Options, _ *workflow.Contribution) (any, error) {
	roles := own(o).Roles
	return Repository{
		Implementation: slices.Contains(roles, Implementation),
		Definition:     slices.Contains(roles, Definition),
	}, nil
}

// Contribution returns the job spec of ci.yml for o, and for the options at the baseline when o is
// not the options of spec. The job checks text, so it runs on the Linux runner of the section
// github, and it keeps spec in the tool cache of GitHub Actions. It has a step for each role:
//
//   - An implementation checks its copies with spec check --remote, which reads the definitions on
//     GitHub with the token of the job.
//   - A definition checks its manifest with spec manifest --check.
//
// Options without a role do not contribute a job.
func (Producer) Contribution(o language.Options) workflow.Contribution {
	opts := own(o)
	steps := make([]workflow.Step, 0, len(opts.Roles))
	if slices.Contains(opts.Roles, Implementation) {
		steps = append(steps, workflow.Step{
			Name: "Check the copies of the definitions",
			Env: map[string]string{ //nolint:gosec // GitHub Actions fills in the token of the job at run time.
				"GITHUB_TOKEN": "${{ github.token }}",
			},
			Run: []string{"ergon tool run " + Name + ".spec -- check --remote"},
		})
	}
	if slices.Contains(opts.Roles, Definition) {
		steps = append(steps, workflow.Step{
			Name: "Check the manifest of the definition",
			Run:  []string{"ergon tool run " + Name + ".spec -- manifest --check"},
		})
	}
	if len(steps) == 0 {
		return workflow.Contribution{}
	}
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:          Name,
		Name:        "Spec",
		Text:        true,
		Timeout:     opts.CI.Timeout,
		Permissions: read,
		Ergon:       true,
		Tools:       true,
		Steps:       steps,
	}}}
}

// own returns o as the options of spec, and the options at the baseline when o is not.
func own(o language.Options) *Options {
	if opts, ok := o.(*Options); ok {
		return opts
	}
	opts, _ := Producer{}.Options().(*Options)
	return opts
}
