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
				Version: "2.20.2",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "f3d24969a7389a46613875711d15a96d8f6ef1553d84c6ace09d69ac1e7d44c0",
					option.LinuxARM64:   "f0fa989c3531348e8f975aac647eb187a09046f8a34c54e0b79bed66c84b7f96",
					option.DarwinAMD64:  "58814536817f6ad787607ed7dfc9e774b3f1334dcffc87002ae2a54fd7b5c5c1",
					option.DarwinARM64:  "4e68482f6bbdb2d3ce7f835be50a4eb7287ee999cefc563c49c5f42ad3778c78",
					option.WindowsAMD64: "d230d6def8b6059b277d472a2790501516b429ffe83b91469daad8b7736b8a5d",
					option.WindowsARM64: "cf152a7d59e3d7d9cf278c7463e3f2b26cc28555e8fdea95c1dcfa773cb385f1",
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
				Version: "1.57.1",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "bf52519eb57f675d47c07d43213cb6dff6449f3d6d1147ff6701ed91f44eef0c",
					option.LinuxARM64:   "aa3d3e07344f590bcce8d31043b752daa76c8a56c2aa34ceeee63c6168152582",
					option.DarwinAMD64:  "2355758c1142982a67b309604c5d75effdbe638a57c771979ea8fd1a16656abc",
					option.DarwinARM64:  "e0a74a5d2688d1254b4e17b71f068a83215e3625f846265bf6614c7ba06f3a96",
					option.WindowsAMD64: "3a01fe865076bb548352195b77e88fa4649277861ea458efe165c3a810b6dcc7",
					option.WindowsARM64: "8cdd580b5f7160d12b66c7c5191100a6cf792e130d8a28deeae5ea45e0030e00",
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
			Analyzers:                 "go.dokimi.dev/lint/golangci@v0.3.0",
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
