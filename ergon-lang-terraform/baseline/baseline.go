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
					option.LinuxAMD64:   "cca9d13e2e1d7a2c627af60ff899a3c9b74212899416aeb96ec764d2ef954537",
					option.DarwinARM64:  "2496e9cb3d24992d553b45e7c87a0fdc9449ca975233876247a9bfeda857e6c0",
					option.WindowsAMD64: "fb42fb859d844b156a8ea9d3363078c4d8b85ca78782e60876b08c9b8e59f303",
				},
				Version: "0.64.0",
			}},
			AWSRuleset: AWSRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "98a530130d33efd49408f3d0ab9198072192ede413cc2a2ce38295d0922eb1c6",
					option.DarwinARM64:  "246818b24c65a60d2b95d7c88921480d72bdca89cfbb936f23bb583b8c631f87",
					option.WindowsAMD64: "ef295f1f773a6ed63be377c192f903e8eb7f811ef603beb2eb21c681bd38c06d",
				},
				Version: "0.49.0",
			}},
			AzureRMRuleset: AzureRMRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "f1150b2dea7aef001ad07002671c75b5495f4521ddf3538ededf00896f7b0aaa",
					option.DarwinARM64:  "f0140f15332e15e710f149d53cf9cbc2d48da051fadd6eec84f52db6094dc43f",
					option.WindowsAMD64: "98025d2ae156aef33272a8520d7425da5f4a70f5e04aac8a7ba999766f0be3d1",
				},
				Version: "0.32.0",
			}},
			GoogleRuleset: GoogleRuleset{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "a5c02b3de937456de2521155831eaf1cc876014a21445bf7dd1621654511fe28",
					option.DarwinARM64:  "93e1a6f2c196e9ba5fa6359cd8d2116d18c1770bddb09bf333e0617e216a5e22",
					option.WindowsAMD64: "cb01c16dbc21c306d3f529cd9352ef006e81c5d42d85eacb581e508eb764df3b",
				},
				Version: "0.40.0",
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
					option.LinuxAMD64:   "f68d291c75783122dad58ab77def0e13f3233f358731db5d6d7e8715603b928f",
					option.DarwinARM64:  "743f85035a1996ecb350659e01cd9faea9709829be09b1b4b03c73eaf3c9cd73",
					option.WindowsAMD64: "a38b8d1902b7cfbc76b423ba1bb456f623f42064cdc72628b6075b89afd1019a",
				},
				Version: "0.13.5",
			}},
			Checkov: "checkov@3.3.30",
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
