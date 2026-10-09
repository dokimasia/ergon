// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// recordEnv names the file into which the test binary, run as ergon by the tools of the toolchain
// of Go, writes its arguments and the version of its environment, one per line. It then writes an
// empty artifacts.json into the dist directory of the configuration of the cases.
const recordEnv = "ERGON_APP_TEST_RECORD"

// The configuration of GoReleaser of the cases, and the artifacts.json of its dist directory.
const (
	config    = "dist: dist/goreleaser/demo\n"
	artifacts = "dist/goreleaser/demo/artifacts.json"
)

// TestMain acts as ergon run by the tools of the toolchain of Go when recordEnv is set. Otherwise it
// isolates git from the configuration of the user and runs the tests.
func TestMain(m *testing.M) {
	if record := os.Getenv(recordEnv); record != "" {
		lines := slices.Concat(os.Args[1:], []string{os.Getenv("ERGON_VERSION")})
		err := os.WriteFile(record, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(artifacts), 0o755)
		}
		if err == nil {
			err = os.WriteFile(artifacts, []byte("[]"), 0o600)
		}
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds every language with the toolchain that builds it", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, app.Register(&c), "Register of every language module")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: csharp.Language, Toolchain: csharp.Toolchain},
				{Name: java.Language, Toolchain: java.Toolchain},
				{Name: kotlin.Language, Toolchain: java.Toolchain},
				{Name: php.Language, Toolchain: php.Toolchain},
				{Name: javascript.Language, Toolchain: javascript.Toolchain},
				{Name: typescript.Language, Toolchain: javascript.Toolchain},
				{Name: golang.Language, Toolchain: golang.Toolchain},
				{Name: python.Language, Toolchain: python.Toolchain},
				{Name: rust.Language, Toolchain: rust.Toolchain},
				{Name: terraform.Language, Toolchain: terraform.Toolchain},
				{Name: bash.Language, Toolchain: bash.Toolchain},
			}, "the languages of the catalog")
		})

		t.Run("returns ErrRegistered for a catalog that has a language of ergon", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, csharp.Register(&c), "Register of C#")
			assert.ErrorIs(t, app.Register(&c), language.ErrRegistered, "Register of every language module")
		})
	})
}

// TestRegisterProcess runs the case whose packer starts the test binary as ergon, alone, because two
// processes of a coverage run that exit in the same nanosecond write one coverage file. It sets the
// environment of the process.
func TestRegisterProcess(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		t.Run("packs a module of Go with GoReleaser through ergon tool run at the tag of the module",
			func(t *testing.T) {
				root := vcstest.Repository(t, files.Tree{".goreleaser.yaml": files.Text(config)})
				head := vcstest.Commit(t, root, "first")
				record := filepath.Join(t.TempDir(), "record")
				t.Setenv(recordEnv, record)
				t.Chdir(root)
				var c language.Catalog
				assert.NoError(t, app.Register(&c), "Register of every language module")
				packer, ok := language.ToolchainRole[language.Packer](&c, golang.Toolchain)
				assert.True(t, ok, "the packer of Go")
				v, err := version.Parse("1.0.0")
				assert.NoError(t, err, "Parse")
				pkgs := []workspace.Package{{Name: "go.dokimi.dev/demo", Dir: ".", Version: v}}
				assert.NoError(t, packer.Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
				got, err := os.ReadFile(record)
				assert.NoError(t, err, "the record of the run")
				assert.Equal(t, string(got), "tool\nrun\ngo.goreleaser\n--\nrelease\n--snapshot\n--clean\n--config\n"+
					".goreleaser.yaml\n1.0.0\n", "the arguments and the version of the run")
				assert.Equal(t, strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "v1.0.0^{commit}")), head,
					"the commit of the tag of the module")
			})
	})
}
