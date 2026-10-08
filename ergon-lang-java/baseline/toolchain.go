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
				option.LinuxAMD64:   "ca69b3d3cd08f889a49dc0a383122f71cc528b83803671df5fd874d97485b108",
				option.DarwinARM64:  "98c460dcd37de25819babd757d04542045b6243113e209edcd4d89fedb0256b4",
				option.WindowsAMD64: "e0ed7644118b717b028c249ee9d3515024e55e8510747ca08906eb96765354d6",
			},
			Version: "2.6.0",
		}}},
		CI: option.MatrixCI[ToolchainActions]{
			Actions: ToolchainActions{SetupJava: workflow.Action{
				Uses:    "actions/setup-java",
				Commit:  "de7274f081f381c8f8158605e0321c36c376e2e6",
				Release: "v6.0.1",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
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
