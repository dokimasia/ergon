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

// Producer renders the files of Go: the configuration of golangci-lint, the configuration of
// GoReleaser of each module with a command, and the fragments of .editorconfig, .gitattributes,
// .gitignore and the Makefile. Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
	_ language.Placer       = Producer{}
)

// Templates returns the templates of Go: .golangci.yml under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section go at the baseline: the releases of golangci-lint, govulncheck,
// benchstat, dokimi-mutate-go, GoReleaser, cosign, syft, UPX and zig, every package of each module, the
// gate of lint, test, race and audit, the nightly jobs of fuzz for 120 minutes, bench for 45 and
// mutate for 60, the options of each step with the analyzers of go.dokimi.dev/lint, no other module
// plugin of golangci-lint and go generate as the generators, no command and no tap, the release of
// setup-go, and a limit of 30 minutes for the job check-go on every runner and the version of
// go.work.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			GolangCILint:   "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
			Govulncheck:    "golang.org/x/vuln/cmd/govulncheck@v1.9.0",
			Benchstat:      "golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e",
			DokimiMutateGo: "go.dokimi.dev/mutate/cmd/dokimi-mutate-go@v0.1.0",
			GoReleaser: GoReleaser{Binary: option.Binary{
				Version: "2.19.0",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "ebc3194ad1d1b1787614405285a0039665cd4c5bd3305215a1b201600f90630b",
					option.LinuxARM64:   "ef33741b684c9f62d6a21e7a3c8b051f139ea7b1d301693b67cc819c46226d53",
					option.DarwinAMD64:  "164fce297e601ec00009e40d26858789b637335b5564bff2c99027b10d28bafe",
					option.DarwinARM64:  "713192b46698ac10ed9275abf25c922a152ab8280c71ae58dc0ff46a33f60004",
					option.WindowsAMD64: "67807490e909a37f58ef40b67601896d8d4da2411c9839221dae55732c243fe9",
					option.WindowsARM64: "ff7a728ebdf86770d45c522f88b901e7fb3c06692b0d0c6a8a3a2c9bcf01b682",
				},
			}},
			Cosign: Cosign{Binary: option.Binary{
				Version: "3.1.4",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "424e2c97d0b45fd7a6b608a87f090f35633f4c23ece5f092d48bba11aaf4bbfa",
					option.LinuxARM64:   "ade83b7aac62e131cacfe96d958948b2150cc9efec46a570a2649b8e27dedad2",
					option.DarwinAMD64:  "9d1e4abe2b6be279ed0d2ff706fde1ccd738747c834e3e6d44039c238867334f",
					option.DarwinARM64:  "9f9b8cd557cf2e5b3df3373a00baa005a86b02391e3e06dd0ce37b3a1533d3a4",
					option.WindowsAMD64: "cbe953787074611f1334cbf37dc1cf953995bb8c45398eb4fc7bb781d29c1f2a",
				},
			}},
			Syft: Syft{Binary: option.Binary{
				Version: "1.55.0",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "bb65249542db5fedeeb4d3f9b018b75e7264dbfbbb10486f044581a92f90490d",
					option.LinuxARM64:   "929bd163db2d7cd2255d3bcb8967208079813d7b46b0c952f7b794d8e2c7d304",
					option.DarwinAMD64:  "e2e5831cbabaec9a36fe65c34755c1113982c842a685edce3ce0b54a0ded8003",
					option.DarwinARM64:  "1217c258bd5eb91ef3189cf22a82e02f9cac07e4ce934350f428e93aff674625",
					option.WindowsAMD64: "ada4c9b93af575f479ce80fce6629897272a2d41f300a14bc49bace020767ca2",
					option.WindowsARM64: "f52f7796e41e4d1ae511c228c99239fd09887077336dcb7a8c2aa7046ed7c816",
				},
			}},
			UPX: UPX{Binary: option.Binary{
				Version: "5.2.1",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "402162aad30af47e60dbd767fb2e64ca394ace9727ba1f40283641f1d1b91657",
					option.LinuxARM64:   "a72d112c5970a904a31da0b9c84f919bc16b9a311787c12245508544a78c7d36",
					option.WindowsAMD64: "eabc6792a347d45e945be7748423e7868fd01b0d2bcaa2f4b1031fd71ff69bda",
				},
			}},
			Zig: option.Zig{
				Version: "0.17.0",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "1cbe9df9f27e6b78d14ccbca43b6703a404ef79ef1c463de901d7f088d4e2026",
					option.LinuxARM64:   "9e8d11661d4ae3bd57702a3832781e23ad151dde5798e16a5ccd503f65234ff8",
					option.DarwinAMD64:  "4f9a1c5269aa17ebda5e6d3c2b89d6cbf36f7d2b22a0306e9ab98f25f95529c6",
					option.DarwinARM64:  "b607e9b9234790a008116ae5bdb71c6243b84b9fb42a53a9e70fde41c06c536a",
					option.WindowsAMD64: "b5663f69581dcf391293fbf16c06cb80d81d806545ce618b4d0bab7f0eb8c428",
					option.WindowsARM64: "0a59d91fa1cb40cf068e9b0954434ce973500c7a2ea749f1e01af62cdab52d26",
				},
			},
		},
		Paths:    option.Paths{"./..."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepRace, option.StepAudit},
		Nightly:  option.Nightly{option.StepFuzz: 120, option.StepBench: 45, option.StepMutate: 60},
		Lint:     Lint{Analyzers: "go.dokimi.dev/lint/golangci@v0.2.0", Plugins: option.Plugins{}},
		Test:     option.Run{Args: []string{"-count=1"}},
		Race:     option.Run{Args: []string{"-count=1", "-p=1"}},
		Fuzz:     option.Fuzz{Match: ".", Time: "30s", Args: []string{"-fuzzminimizetime=5s"}},
		Bench:    option.Bench{Match: ".", Time: "1s", Args: []string{"-benchmem"}, Skip: []string{}, Count: 6},
		Mutate:   option.Mutate{Timeout: "0s", Args: []string{}, Skip: []string{}, Workers: 1},
		Generate: option.Generate{Commands: [][]string{{"go", "generate"}}},
		Audit:    option.Run{Args: []string{}},
		Binaries: []Command{},
		Homebrew: Homebrew{Tap: ""},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupGo: workflow.Action{
				Uses:    "actions/setup-go",
				Commit:  "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
				Release: "v7.0.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
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
