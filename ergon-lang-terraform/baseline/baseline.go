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
					option.LinuxAMD64:   "d4c545ed2e29d6338604b586f34f544f380f70f243aa72f7b484713835eaec61",
					option.DarwinARM64:  "5a71c63481ba56760864298011fdb7317a9ae80a0bbaeceefa35883c7f62ac71",
					option.WindowsAMD64: "292e760adf0e9c7e0f10ddeb0f6add67fd6d2c68b5449fdf766ff6ffddce465c",
				},
				Version: "0.65.0",
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
					option.LinuxAMD64:   "9005daf969de0b50134493a2c00078b49f5f5b39d021cda7c89bf4d4f3d776d3",
					option.DarwinARM64:  "f6b114f4b032f3f9202ab6c23bfd28c3c8e68aeeb8a8f12fc118bf2073081d71",
					option.WindowsAMD64: "afc02cbdf63726d3e5fb26a077ca1f24acd4820d7e44c21b03a3e030f9266490",
				},
				Version: "0.24.0",
			}},
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "f29c3e94ffabd7e6f3ba521dba169a9b7ec12100bf3257087f0f24f6123f3645",
					option.DarwinARM64:  "60fbd5c8a1ed681845eea28d9d622fcde03c51e91a43465b64a1e17db1ed3472",
					option.WindowsAMD64: "a2a8820d5eea32f08ef467ef3b7f66883a9b076d5737a2fc2fbb5f458d1bb50f",
				},
				Version: "0.13.6",
			}},
			Checkov: "checkov@3.4.2",
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
