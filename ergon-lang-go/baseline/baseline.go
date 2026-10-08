// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"embed"
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// Name is the name of the producer of Go in the lock, and of its section of .ergon.yaml, which is
// the name of the language.
const Name = "go"

// templates are the templates of Go: the configuration of golangci-lint under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of Go: the configuration of golangci-lint, and the fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use, and
// it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Go: .golangci.yml under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section go at the baseline: the releases of golangci-lint, govulncheck,
// benchstat, dokimi-mutate-go and ergon-go-vet, every package of each module, the gate of lint,
// test, race and audit, the options of each step with go generate as the generators, the release of
// setup-go, and a limit of 30 minutes for the job check-go on every runner and the version of
// go.work.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			GolangCILint:   "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0",
			Govulncheck:    "golang.org/x/vuln/cmd/govulncheck@v1.8.0",
			Benchstat:      "golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68",
			DokimiMutateGo: "go.dokimi.dev/mutate/cmd/dokimi-mutate-go@v0.0.0-20261006212535-719083ce3457",
			ErgonGoVet:     "go.dokimi.dev/ergon/lang/go/cmd/ergon-go-vet@v0.1.0",
		},
		Paths:    option.Paths{"./..."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepRace, option.StepAudit},
		Lint:     Lint{Exclude: option.Paths{}},
		Test:     option.Run{Args: []string{"-count=1"}},
		Race:     option.Run{Args: []string{"-count=1", "-p=1"}},
		Fuzz:     option.Fuzz{Match: ".", Time: "30s", Args: []string{"-fuzzminimizetime=5s"}},
		Bench:    option.Bench{Match: ".", Time: "1s", Args: []string{"-benchmem"}, Count: 6},
		Mutate:   option.Mutate{Timeout: "0s", Args: []string{}, Workers: 1},
		Generate: option.Generate{Command: []string{"go", "generate"}, Args: []string{}},
		Audit:    option.Run{Args: []string{}},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupGo: workflow.Action{
				Uses:    "actions/setup-go",
				Commit:  "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
				Release: "v7.0.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of Go of the workflows for o, as [Options.Contribution] states it,
// and for the options at the baseline when o is not the section go.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
