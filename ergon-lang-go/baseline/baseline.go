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

// uncached is the flag of go test that runs each test once and reads no result from the cache of go
// test, which the steps test, race and longtest pass at the baseline.
const uncached = "-count=1"

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
// plugin of golangci-lint, a new cache of golangci-lint every 7 days and go generate as the
// generators, no command and no tap, the release of setup-go, and a limit of 30 minutes for the job
// check-go on every runner and the version of go.work.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			GolangCILint:   "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
			Govulncheck:    "golang.org/x/vuln/cmd/govulncheck@v1.9.0",
			Benchstat:      "golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e",
			DokimiMutateGo: "go.dokimi.dev/mutate/cmd/dokimi-mutate-go@v0.3.0",
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
				Version: "5.3.0",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "79cea8a50b6fa102d10c7b79a419cd77dead3cc2d78661de2dc5e5815dc505c1",
					option.LinuxARM64:   "2a4afa860ac7aa6968f88df7841823746e5d00ba56affdf9342c2f5254fb776f",
					option.WindowsAMD64: "d127996c76615e431be810e9c8d2910dbe3dd400600c5c0c50276fc5a17a9705",
				},
			}},
			Zig: option.Zig{
				Version: "0.17.1",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "3581863e3cbcfd0fce0efc682ce7e9e93da96668a977791d356a26d2a7e9b17a",
					option.LinuxARM64:   "a1546d5646eb184437918d6635ca6b762a0e889eac09cef720ee7a1ed3fe7795",
					option.DarwinAMD64:  "9af07b20319fbd0c92ee46ad23e81d6a782b1aaea2cfb8dcbff35f1d2c7ddbb7",
					option.DarwinARM64:  "75f1bc7a1e9852f17ec9a0c1895f767c4ec56d46bc8a834920eece1d0359dd8d",
					option.WindowsAMD64: "b03f477a1943673e02aca0f570e8030dffd536ce1f45cbc4d3326ec31159562e",
					option.WindowsARM64: "4789895bdb1e312a2ea73b2a7c0a0cc8d2ab62ba6976e834a7c54cf443377c15",
				},
			},
		},
		Paths:   option.Paths{"./..."},
		Check:   option.Check{option.StepLint, option.StepTest, option.StepRace, option.StepAudit},
		Nightly: option.Nightly{option.StepFuzz: 120, option.StepBench: 45, option.StepMutate: 60},
		Lint: Lint{
			Analyzers:                 "go.dokimi.dev/lint/golangci@v0.2.0",
			Plugins:                   option.Plugins{},
			CacheInvalidationInterval: 7,
		},
		Test:     option.Run{Args: []string{uncached}},
		Race:     option.Run{Args: []string{uncached, "-p=1"}},
		LongTest: option.Run{Args: []string{uncached}},
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
