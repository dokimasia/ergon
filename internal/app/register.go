// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	gobaseline "go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	jsbaseline "go.dokimi.dev/ergon/lang/javascript/baseline"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	pybaseline "go.dokimi.dev/ergon/lang/python/baseline"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/vcs"
)

// git is the access to the repository that the toolchain of Go reads tags and snapshots through,
// and creates the tag of a module with commands through.
var git = golang.Git{Tags: vcs.Tags, Snapshot: vcs.Snapshot, Head: vcs.Head, LightTag: vcs.LightTag}

// repository is the access to the repository that the toolchain of Terraform reads the tags and the
// answers of the lock of ergon init through.
var repository = terraform.Repository{Tags: vcs.Tags, Answers: answers}

// registryTimeout is the limit of a request of a toolchain to its registry, such as the check of
// PyPI, of the npm registry or of the index of crates.io for a release.
const registryTimeout = time.Minute

// registrations returns the Register functions of the language modules, in the order that
// [Register] calls them, with client as the client of the requests to the registries. Java precedes
// Kotlin and JavaScript precedes TypeScript, because Kotlin and TypeScript name the toolchains that
// Java and JavaScript register.
func registrations(client *http.Client) []func(*language.Catalog) error {
	return []func(*language.Catalog) error{
		csharp.Register,
		java.Register,
		kotlin.Register,
		php.Register,
		func(c *language.Catalog) error {
			jsTools := javascript.Tools{Run: tools(jsbaseline.ToolchainName), Command: command}
			return javascript.Register(c, jsTools, client)
		},
		typescript.Register,
		func(c *language.Catalog) error {
			goTools := golang.Tools{Run: tools(gobaseline.Name), Program: programs(gobaseline.Name)}
			return golang.Register(c, git, goTools)
		},
		func(c *language.Catalog) error { return python.Register(c, tools(pybaseline.Name), client) },
		func(c *language.Catalog) error {
			return rust.Register(c, rust.Tools{Command: command, Getenv: os.Getenv}, client)
		},
		func(c *language.Catalog) error { return terraform.Register(c, repository) },
		bash.Register,
	}
}

// Register adds the toolchains and the languages of ergon to c, one language module at a time. The
// toolchains send their requests to a registry through transport, such as the check of PyPI, of the
// npm registry or of the index of crates.io for a release, each within registryTimeout. The
// toolchain of Rust reads the variables of the OIDC token of a job of GitHub Actions from the
// environment of the process. Register stops at the first error of a module's Register and
// returns it, such as an error that wraps [language.ErrRegistered] for a catalog that already has
// one of the names.
func Register(c *language.Catalog, transport http.RoundTripper) error {
	client := &http.Client{Transport: transport, Timeout: registryTimeout}
	for _, register := range registrations(client) {
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

// command runs the program name of the PATH of the process with args in dir, with env added to the
// environment of the process. The program writes its output to the standard error of the process,
// so the log of a release shows the output of the program, such as npm pack. command returns the
// error of the program, such as an exit status other than 0.
func command(ctx context.Context, dir, name string, args, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("app: %s: %w", name, err)
	}
	return nil
}

// answers returns the answers of the lock of ergon init of the repository at dir, and the zero
// Answers for a repository without a lock. It returns the error of reading the lock, and the error
// of lock.Decode, which wraps lock.ErrInvalid.
func answers(_ context.Context, dir string) (language.Answers, error) {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(lock.Path)))
	if errors.Is(err, fs.ErrNotExist) {
		return language.Answers{}, nil
	}
	if err != nil {
		return language.Answers{}, fmt.Errorf("app: read %s: %w", lock.Path, err)
	}
	l, err := lock.Decode(data)
	if err != nil {
		return language.Answers{}, err
	}
	return l.Answers, nil
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
