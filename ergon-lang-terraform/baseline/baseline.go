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

// Options returns the section terraform at the baseline: the releases of tflint and uv with the
// digests that the releases state, the release of checkov, the root of the repository, the gate of
// lint, test and audit, the release of setup-terraform, and a limit of 30 minutes for the job
// check-terraform on every runner and the version of .terraform-version.
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
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "9167d72b3319674b6303c4cbe071854bba13ebdf3d76b1a7cbdc175471fb66d6",
					option.DarwinARM64:  "50487ae565ccd96e499056b4674d438f4c53170202617b4c759defe0c6a1b544",
					option.WindowsAMD64: "75d05de6762778c31ee183398de7dd15093fad0ed90b1f236d8205ea5ec00c90",
				},
				Version: "0.12.23",
			}},
			Checkov: "checkov@3.3.23",
		},
		Paths: option.Paths{"."},
		Check: option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:  option.Run{Args: []string{}},
		Audit: option.Audit{Ignore: []string{}},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupTerraform: workflow.Action{
				Uses:    "hashicorp/setup-terraform",
				Commit:  "dfe3c3f87815947d99a8997f908cb6525fc44e9e",
				Release: "v4.0.1",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
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
