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

// Name is the name of the producer of Terraform in the lock, and of its section of .ergon.yaml,
// which is the name of the language.
const Name = "terraform"

// templates are the templates of Terraform: the configuration of tflint under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of Terraform: the configuration of tflint, and the fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use, and
// it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Terraform: .tflint.hcl under managed/, and the fragments of
// the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section terraform at the baseline: the releases of tflint, of its rulesets of
// AWS, Azure and Google Cloud, of terraform-docs and of uv with the digests that the releases state,
// the release of checkov, the root of the repository, the gate of lint, test and audit, no ruleset
// that lint-terraform loads, no generators, the release of setup-terraform, and a limit of 30
// minutes for the job check-terraform on every runner and the version of .terraform-version.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			TFLint: TFLint{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "22be11cadc25cd02401e94c0ccc91aa63467debb8485c2a755d98f4c0a9d1e2e",
					option.DarwinARM64:  "859b26af07ead7e65cee7eec9d050b2e2571dc28395b7a39191cae088cd67803",
					option.WindowsAMD64: "e3acd1d89c735ce6e09108c9277f26076ebeb385c1f77a73c1cbf1826ed626a7",
				},
				Version: "0.66.0",
			}},
			AWSRuleset: AWSRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "cdd433fdbc0929e7b9f9709dd9e72237fdfe1fc235674445c284b1e4b94a5c26",
					option.DarwinARM64:  "0117f75027cbc1bc452db1defd59d40b4a579ef3e7bb3a8b3079eaf6d372a9a6",
					option.WindowsAMD64: "42cb38eeef91b13b2df1fd3432685fbbff7d0f8850248beab08b0b42567ae933",
				},
				Version: "0.50.0",
			}},
			AzureRMRuleset: AzureRMRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "cd372389bd9ef17096ed7a9fbd856c0122521df141301685e974b269322819a8",
					option.DarwinARM64:  "af434684e202bdac741bff19425417b2848f807624be130877a0e3fe7a1ed531",
					option.WindowsAMD64: "367ead184ec4ea6cdf4c401607c03301a3824a514815def94d1b378aae357630",
				},
				Version: "0.33.0",
			}},
			GoogleRuleset: GoogleRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "efb3910bbc9f0b90bebd76d475ca6cd2cf04a032f6cddc8fa7a334e5011da6d6",
					option.DarwinARM64:  "bcfaa5613f70632c5a04540775c88552d5ab9ba5c2bbb5e46fc03b4be3ae630a",
					option.WindowsAMD64: "7e5e3f8b165bd2b65bf378d70fd4fc15117ca0bc7898ab5953671570798c3620",
				},
				Version: "0.41.0",
			}},
			TerraformDocs: TerraformDocs{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "246f1a3211c9dbbed4c4af82b856dbd40acda78816ac8cdb5203c2869b659c76",
					option.DarwinARM64:  "b2d2c6a618f331cacb2f766102a8e917fec758bca9573e3e539a799ae4bce225",
					option.WindowsAMD64: "b9d1ceaf9136fcf4ddce55a5590cc4fca19ac349542c51c28b8dfa642ed85fa8",
				},
				Version: "0.25.1",
			}},
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "5acfd9177cfecf0bf87578f059728215ccc884388988f55f7a65041b57387d91",
					option.DarwinARM64:  "6b71028a6b904f2ae3340ef52e595512de807d06c0a20eaef2f692581b246558",
					option.WindowsAMD64: "529e78a9cec6cd7e3e014f7ecd8bda0c113969d4ed0f2efeedb85c30943325b8",
				},
				Version: "0.13.8",
			}},
			Checkov: "checkov@3.4.5",
		},
		Paths:    option.Paths{"."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Lint:     Lint{Rulesets: []Ruleset{}},
		Test:     option.Run{Args: []string{}},
		Generate: Generate{Commands: [][]string{}},
		Audit:    option.Audit{Ignore: []string{}},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupTerraform: workflow.Action{
				Uses:    "hashicorp/setup-terraform",
				Commit:  "dfe3c3f87815947d99a8997f908cb6525fc44e9e",
				Release: "v4.0.1",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of Terraform of the workflows for o, as [Options.Contribution]
// states it, and for the options at the baseline when o is not the section terraform.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
