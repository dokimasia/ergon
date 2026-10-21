// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	gobaseline "go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	pybaseline "go.dokimi.dev/ergon/lang/python/baseline"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
	"go.dokimi.dev/ergon/service/vcs"
)

// git is the access to the repository that the toolchain of Go reads tags and snapshots through,
// and creates the tag of a module with commands through.
var git = golang.Git{Tags: vcs.Tags, Snapshot: vcs.Snapshot, Head: vcs.Head, LightTag: vcs.LightTag}

// registrations are the Register functions of the language modules, in the order that [Register]
// calls them. Java precedes Kotlin and JavaScript precedes TypeScript, because Kotlin and
// TypeScript name the toolchains that Java and JavaScript register.
var registrations = []func(*language.Catalog) error{
	csharp.Register,
	java.Register,
	kotlin.Register,
	php.Register,
	javascript.Register,
	typescript.Register,
	func(c *language.Catalog) error {
		return golang.Register(c, git, golang.Tools{Run: tools(gobaseline.Name), Program: programs(gobaseline.Name)})
	},
	func(c *language.Catalog) error { return python.Register(c, tools(pybaseline.Name)) },
	rust.Register,
	terraform.Register,
	bash.Register,
}

// Register adds the toolchains and the languages of ergon to c, one language module at a time.
// It stops at the first error of a module's Register and returns it, such as an error that wraps
// [language.ErrRegistered] for a catalog that already has one of the names.
func Register(c *language.Catalog) error {
	for _, register := range registrations {
		if err := register(c); err != nil {
			return err
		}
	}
	return nil
}

// tools returns a function that runs the tool of the section of .ergon.yaml with args in dir
// through ergon tool run of the running ergon, with env added to the environment of the process and
// the standard streams of the process, so the log of a release shows the output of the tool, such
// as GoReleaser or uv. The function returns the error of finding the running ergon, and the error
// of the tool, such as an exit status other than 0.
func tools(section string) func(ctx context.Context, dir, tool string, args, env []string) error {
	return func(ctx context.Context, dir, tool string, args, env []string) error {
		self, err := os.Executable()
		if err == nil {
			argv := slices.Concat([]string{"tool", "run", section + "." + tool, "--"}, args)
			cmd := exec.CommandContext(ctx, self, argv...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), env...)
			cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
			err = cmd.Run()
		}
		if err != nil {
			return fmt.Errorf("app: ergon tool run %s.%s: %w", section, tool, err)
		}
		return nil
	}
}

// programs returns a function that installs the tool of the section of .ergon.yaml for dir through
// ergon tool path of the running ergon, unless the cache has it, and returns the program that
// ergon tool path writes, such as zig. The standard error of the process receives the output of
// an install. The function returns the error of finding the running ergon, and the error of ergon
// tool path, such as an exit status other than 0.
func programs(section string) func(ctx context.Context, dir, tool string) (string, error) {
	return func(ctx context.Context, dir, tool string) (string, error) {
		self, err := os.Executable()
		var out []byte
		if err == nil {
			cmd := exec.CommandContext(ctx, self, "tool", "path", section+"."+tool)
			cmd.Dir = dir
			cmd.Stderr = os.Stderr
			out, err = cmd.Output()
		}
		if err != nil {
			return "", fmt.Errorf("app: ergon tool path %s.%s: %w", section, tool, err)
		}
		return strings.TrimSuffix(string(out), "\n"), nil
	}
}
