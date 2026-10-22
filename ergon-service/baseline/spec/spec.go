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

// templates are the templates of spec, under shared/.
//
//go:embed all:templates
var templates embed.FS

// The permissions of the jobs of spec.
var (
	// read is the permission of the job spec to read the contents of the repository.
	read = map[string]string{"contents": "read"}

	// write is the permission of the job propose-spec to write the branches and the pull requests
	// of its proposals.
	write = map[string]string{"contents": "write", "pull-requests": "write"}
)

// Repository are the data of the templates of spec: what the repository is to spec, which
// [Producer.Data] computes from the roles of the options.
type Repository struct {
	// Implementation reports that the repository has copies of definitions.
	Implementation bool

	// Definition reports that the repository is the repository of a definition.
	Definition bool
}

// Producer renders the files of spec in a repository with a role: the targets of spec in the
// Makefile, the job spec of ci.yml, and the job propose-spec of nightly.yml in an implementation.
// Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Calculator   = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of spec: the fragment of the Makefile with the targets of spec
// under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the options of spec at the baseline: no role, the release 0.4.0 of spec with the
// digests that it states for every platform, and a limit of 10 minutes for each job of spec.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{Spec: Spec{Binary: option.Binary{
			SHA256: map[option.Platform]string{
				option.LinuxAMD64:   "52c6e38ae99491e5cbeaf5e48ffb7aeec7a4ab432a9f925d5f63788cdafdfba6",
				option.LinuxARM64:   "533883b706ebb63501d3791f37e74d6b50dfa5707b59e641ce6415dea5af6443",
				option.DarwinAMD64:  "af667806ff4d1b98975dbf25d9994748edc9abc0c8b170ea478365fe86d7ace3",
				option.DarwinARM64:  "ab07badc7456450212860080aa8b4c17b490b7bf94578c1f552307b6cfeb287b",
				option.WindowsAMD64: "d3ca18f4ec1991867e6bf3ecad3ad963754f1c8deae0cfb42fd46d42552a39f2",
				option.WindowsARM64: "fcbeb31e10146308d86f2682016f70fd7eb2a4b76ba2f72a4ced9118e9976c88",
			},
			Version: "0.4.0",
		}}},
		Roles: []Role{},
		CI:    option.CI[struct{}]{Timeout: 10},
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

// Contribution returns the jobs of spec for o, and for the options at the baseline when o is not
// the options of spec. Both jobs keep spec in the tool cache of GitHub Actions.
//
// The job spec of ci.yml checks text, so it runs on the Linux runner of the section github. It has
// a step for each role:
//
//   - An implementation checks its copies with spec check --remote, which reads the definitions on
//     GitHub with the token of the job.
//   - A definition checks its manifest with spec manifest --check.
//
// An implementation also gets the job propose-spec of nightly.yml. spec propose opens or updates a
// pull request for each copy that differs from its definition. It closes the pull request of a copy
// that matches. The job passes [workflow.AppToken] to spec as GITHUB_TOKEN. GitHub starts the
// checks of these pull requests without the approval of a maintainer when the repository has the
// GitHub App of ergon.
//
// Options without a role do not contribute a job.
func (Producer) Contribution(o language.Options) workflow.Contribution {
	opts := own(o)
	steps := make([]workflow.Step, 0, len(opts.Roles))
	var nightly []workflow.Job
	if slices.Contains(opts.Roles, Implementation) {
		steps = append(steps, workflow.Step{
			Name: "Check the copies of the definitions",
			Env: map[string]string{ //nolint:gosec // GitHub Actions fills in the token of the job at run time.
				"GITHUB_TOKEN": "${{ github.token }}",
			},
			Run: []string{"ergon tool run " + Name + ".spec -- check --remote"},
		})
		nightly = []workflow.Job{{
			ID:          "propose-" + Name,
			Name:        "Propose spec",
			Timeout:     opts.CI.Timeout,
			Permissions: write,
			Ergon:       true,
			Tools:       true,
			App:         true,
			Steps: []workflow.Step{{
				Name: "Propose the updates of the copies",
				Env:  map[string]string{"GITHUB_TOKEN": workflow.AppToken},
				Run:  []string{"ergon tool run " + Name + `.spec -- propose --repo "$GITHUB_REPOSITORY"`},
			}},
		}}
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
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          Name,
			Name:        "Spec",
			Text:        true,
			Timeout:     opts.CI.Timeout,
			Permissions: read,
			Ergon:       true,
			Tools:       true,
			Steps:       steps,
		}},
		Nightly: nightly,
	}
}

// own returns o as the options of spec, and the options at the baseline when o is not.
func own(o language.Options) *Options {
	if opts, ok := o.(*Options); ok {
		return opts
	}
	opts, _ := Producer{}.Options().(*Options)
	return opts
}
