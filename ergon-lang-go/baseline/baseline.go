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
// benchstat, dokimi-mutate-go, ergon-go-vet, GoReleaser, cosign, syft and UPX, every package of each
// module, the gate of lint, test, race and audit, the nightly jobs of fuzz for 120 minutes, bench
// for 45 and mutate for 60, the options of each step with no module plugin of golangci-lint and go
// generate as the generators, no command and no tap, the release of setup-go, and a limit of 30
// minutes for the job check-go on every runner and the version of go.work.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			GolangCILint:   "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0",
			Govulncheck:    "golang.org/x/vuln/cmd/govulncheck@v1.8.0",
			Benchstat:      "golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68",
			DokimiMutateGo: "go.dokimi.dev/mutate/cmd/dokimi-mutate-go@v0.0.0-20261006212535-719083ce3457",
			ErgonGoVet:     "go.dokimi.dev/ergon/lang/go/cmd/ergon-go-vet@v0.2.1",
			GoReleaser: GoReleaser{Binary: option.Binary{
				Version: "2.18.2",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "0a96edc9d9bc594e4a41cc4d59467c182062910ab24d9d1f6dd7b667d32606d3",
					option.LinuxARM64:   "a71681b29194f08f057a68cfcaa5c6b15d907a83a2622c51900c4faff828f322",
					option.DarwinAMD64:  "5e97d6517f73a0b6f71675a2911b15d3136c03c8907650c2b18e114b1c0f5205",
					option.DarwinARM64:  "a811ff154fe136a0cfb55d00126c151fc39ec370a663d805a9ca5547445aa70c",
					option.WindowsAMD64: "de61a8e7a064abb14210b16c942facd299643d83341c5cd293f19155a7c8b95f",
					option.WindowsARM64: "d2aebc59bee96078c1f529160f170e02ffc96fd5d31b5e33eef66cdd7256f602",
				},
			}},
			Cosign: Cosign{Binary: option.Binary{
				Version: "3.1.3",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "4629c757b7618056f8ddd7e2625ae9fdd94c0372a65049520bc7d9df9efc7f71",
					option.LinuxARM64:   "c5d324e091826b0d7a78eb16fef316450b4eb9aaec045611c08ba06f5e73220a",
					option.DarwinAMD64:  "2347488e5d5b25336644024dfeca5601b190e91197a71a917bda44744aff106c",
					option.DarwinARM64:  "5cf948c2f4dfe59687bdd0b8523709067383e03982cc543475c8a7dc70e92a76",
					option.WindowsAMD64: "9fe59be0eca1271873ce019061335eb1ac419b7059202e797828467ddabe33be",
				},
			}},
			Syft: Syft{Binary: option.Binary{
				Version: "1.54.1",
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "c069905b391cc4c20a5ba65ad5c10be2a7ba074f8ea6ad203e24d14e303dad47",
					option.LinuxARM64:   "dfdf0537610113edbefe1f1fc6548bc957b2d77439636ec824fcf0e10d46d054",
					option.DarwinAMD64:  "2956322838b2f64e470eea474495f0cd96be4f222b1ac037258cdc47d965064e",
					option.DarwinARM64:  "b4319c3abaa87a0170ab76ee83ea2260ca34b53aecfa3ab0dd5428d2319d744f",
					option.WindowsAMD64: "8b56e8285e295e0bbed26eeea9b16ed51c493be97ccdf42dae6326c84fe8e19f",
					option.WindowsARM64: "440019acac7c5b3b44edb8d224aa226b52aadca66a1c3a0a8d58a9525f675e36",
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
		},
		Paths:    option.Paths{"./..."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepRace, option.StepAudit},
		Nightly:  option.Nightly{option.StepFuzz: 120, option.StepBench: 45, option.StepMutate: 60},
		Lint:     Lint{Plugins: option.Plugins{}, Exclude: option.Paths{}},
		Test:     option.Run{Args: []string{"-count=1"}},
		Race:     option.Run{Args: []string{"-count=1", "-p=1"}},
		Fuzz:     option.Fuzz{Match: ".", Time: "30s", Args: []string{"-fuzzminimizetime=5s"}},
		Bench:    option.Bench{Match: ".", Time: "1s", Args: []string{"-benchmem"}, Count: 6},
		Mutate:   option.Mutate{Timeout: "0s", Args: []string{}, Workers: 1},
		Generate: option.Generate{Command: []string{"go", "generate"}, Args: []string{}},
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
