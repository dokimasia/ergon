// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// ToolchainName is the name of the producer of the jvm toolchain in the lock, and of its section of
// .ergon.yaml, which is the name of the toolchain.
const ToolchainName = "jvm"

// Toolchain renders the files that Java and Kotlin share, once for a repository with either or
// both: the fragments of .gitignore and of the Makefile with the target audit-jvm. Its zero value
// is ready to use, and it is safe for concurrent use.
type Toolchain struct{}

var (
	_ language.Producer     = Toolchain{}
	_ language.Configurable = Toolchain{}
	_ language.Contributor  = Toolchain{}
)

// Templates returns the templates of the jvm toolchain: the fragments of the shared files under
// shared/.
func (Toolchain) Templates() fs.FS {
	// templates has the directory templates/jvm, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates/"+ToolchainName)
	return sub
}

// Options returns the section jvm at the baseline: the release of osv-scanner with the digests of
// its assets, the release of setup-java, and a limit of 30 minutes for the jobs of Java and Kotlin
// on every runner and the Java of .java-version.
func (Toolchain) Options() language.Options {
	return &ToolchainOptions{
		Tools: ToolchainTools{OSVScanner: OSVScanner{Binary: option.Binary{
			SHA256: map[option.Platform]string{
				option.LinuxAMD64:   "160c6253e2922e9e0a6a9771c5c4c40a4b3c6090b34a3cdd5e80c96cc2e4218b",
				option.DarwinARM64:  "d4926ef8da134332543be276dff157e4c170f99ad91fc48e89448a8816bdf70b",
				option.WindowsAMD64: "9964ed64ec9a387391eb4cc006ad5c531154821aef29b793e02c448de4686438",
			},
			Version: "2.7.0",
		}}},
		CI: option.MatrixCI[ToolchainActions]{
			Actions: ToolchainActions{SetupJava: workflow.Action{
				Uses:    "actions/setup-java",
				Commit:  "b843ebf9b913146b38da2eaa7aeddf76074ae1ad",
				Release: "v6.1.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of the jvm toolchain of the workflows for o, as
// [ToolchainOptions.Contribution] states it, and for the options at the baseline when o is not the
// section jvm.
func (t Toolchain) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*ToolchainOptions)
	if !ok {
		opts, _ = t.Options().(*ToolchainOptions)
	}
	return opts.Contribution()
}
