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
				option.LinuxAMD64:   "fb7dab19a8bd9bbe5c3189f09d4b7ee1820c9553a23bac202648a56efd2f6ee8",
				option.DarwinARM64:  "b22c02877a811c8f78956e8643efa554932b7378f590d13f8e97d4e9448a29f3",
				option.WindowsAMD64: "7121dcf420aefbbf89a7f0050c4d3094e079ff93afd334dfc96f16b02ec85800",
			},
			Version: "2.7.1",
		}}},
		CI: option.MatrixCI[ToolchainActions]{
			Actions: ToolchainActions{SetupJava: workflow.Action{
				Uses:    "actions/setup-java",
				Commit:  "d1374f8ac74474200d16ef8ecee22c7ae7a51880",
				Release: "v6.3.0",
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
