// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
)

// tap is the Homebrew tap of the cases.
const tap = "dokimasia/homebrew-tap"

// The configurations of GoReleaser of the cases: of the root module, and of the module lint.
const (
	rootConfig = ".goreleaser.yaml"
	lintConfig = "lint/.goreleaser.yaml"
)

// oldestMacOS matches the oldest macOS that the linker of Go writes into a binary, in the source of
// the linker, with the major and the minor version as its groups.
var oldestMacOS = regexp.MustCompile(`\bmacOS\s*=\s*macVersionFlag\{(\d+), (\d+), \d+\}`)

func TestGoReleaser(t *testing.T) {
	t.Parallel()

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Files", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the configuration of each module with a command", func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = commands()
				o.Homebrew.Tap = tap
				files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), o,
					&workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Length(t, files, 2, "the configurations")
				assert.Equal(t, files[0].Path, rootConfig, "the path of the configuration of the root module")
				assert.Equal(t, files[1].Path, lintConfig, "the path of the configuration of the module lint")
				for _, f := range files {
					golden.Match(t, path.Join("goreleaser", f.Path), f.Content, golden.ShouldUpdate())
				}
			})

			t.Run("renders the builds of a command with cgo", func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = []baseline.Command{{
					Name: "eidos", Module: ".", Main: "./cmd/eidos", Description: "Generates code.", CGO: true,
				}}
				files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), o,
					&workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Length(t, files, 1, "the configurations")
				golden.Match(t, path.Join("goreleaser", "cgo", files[0].Path), files[0].Content, golden.ShouldUpdate())
			})

			t.Run("renders the release of a provider of the Terraform Registry", func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = []baseline.Command{providerCommand()}
				a := baselinetest.Answers(golang.Language)
				a.Repository = "example/terraform-provider-demo"
				files, err := baseline.Producer{}.Files(a, o, &workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Length(t, files, 1, "the configurations")
				golden.Match(t, path.Join("goreleaser", "provider", files[0].Path), files[0].Content,
					golden.ShouldUpdate())
			})

			t.Run("returns ErrInvalid for a provider whose name is not the name of the repository", func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = []baseline.Command{providerCommand()}
				a := baselinetest.Answers(golang.Language)
				a.Repository = "example/terraform-provider-other"
				_, err := baseline.Producer{}.Files(a, o, &workflow.Contribution{})
				assert.ErrorIs(t, err, option.ErrInvalid, "Files")
				assert.Contains(t, err.Error(), "the provider terraform-provider-demo has another name than the "+
					"repository example/terraform-provider-other", "the error")
			})

			t.Run("builds the macOS targets of a command with cgo for the oldest macOS of the Go toolchain",
				func(t *testing.T) {
					t.Parallel()
					goroot, err := exec.CommandContext(t.Context(), "go", "env", "GOROOT").Output()
					assert.NoError(t, err, "go env GOROOT")
					linker := files.Read(t, filepath.Join(strings.TrimSpace(string(goroot)), "src", "cmd", "link",
						"internal", "ld", "macho.go"))
					oldest := oldestMacOS.FindStringSubmatch(linker)
					assert.Length(t, oldest, 3, "the oldest macOS that the linker of Go writes")
					o := goOptions()
					o.Binaries = []baseline.Command{{
						Name: "worker", Module: ".", Main: "./cmd/worker", Description: "Runs the jobs.", CGO: true,
					}}
					rendered, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), o,
						&workflow.Contribution{})
					assert.NoError(t, err, "Files")
					assert.Length(t, rendered, 1, "the configurations")
					assert.Contains(t, string(rendered[0].Content), "-macos."+oldest[1]+"."+oldest[2]+"'",
						"the target of zig of the macOS builds")
				})

			t.Run("renders no configuration without a command", func(t *testing.T) {
				t.Parallel()
				files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), goOptions(),
					&workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Empty(t, files, "the configurations")
			})

			t.Run("renders no configuration for options of another type", func(t *testing.T) {
				t.Parallel()
				files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), nil,
					&workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Empty(t, files, "the configurations")
			})

			splits := []struct {
				name      string
				platforms []option.Platform
				cgo       bool
				want      string
				absent    string
			}{
				{
					name:      "packs a command with Linux targets alone in the build -upx",
					platforms: []option.Platform{option.LinuxAMD64},
					want:      "\n    ids: [worker-upx]\n",
					absent:    "\n  - id: worker\n    dir: .\n",
				},
				{
					name:      "renders no build -upx for a command without a Linux target",
					platforms: []option.Platform{option.DarwinARM64, option.WindowsAMD64},
					want:      "\n    ids: [worker]\n",
					absent:    "go.upx",
				},
				{
					name:      "builds the macOS targets of a command with cgo in the build -darwin",
					platforms: []option.Platform{option.DarwinARM64, option.WindowsAMD64},
					cgo:       true,
					want:      "\n    ids: [worker, worker-darwin]\n",
					absent:    "go.upx",
				},
				{
					name:      "renders no build -darwin for a command with cgo without a macOS target",
					platforms: []option.Platform{option.LinuxAMD64, option.WindowsAMD64},
					cgo:       true,
					want:      "\n    ids: [worker, worker-upx]\n",
					absent:    "\n  - id: worker-darwin\n",
				},
			}
			for _, tt := range splits {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					o := goOptions()
					o.Binaries = []baseline.Command{{
						Name: "worker", Module: ".", Main: "./cmd/worker", Description: "Runs the jobs.",
						Platforms: tt.platforms, CGO: tt.cgo,
					}}
					files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), o,
						&workflow.Contribution{})
					assert.NoError(t, err, "Files")
					assert.Length(t, files, 1, "the configurations")
					assert.Contains(t, string(files[0].Content), tt.want, "the builds of the archive")
					assert.NotContains(t, string(files[0].Content), tt.absent, "the configuration")
				})
			}

			t.Run("writes the license of a command over the license of the repository", func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = []baseline.Command{{
					Name: "worker", Module: ".", Main: "./cmd/worker", Description: "Runs the jobs.",
					License: spdx.ID("BUSL-1.1"), Packages: []baseline.Package{baseline.PackageDeb},
				}}
				files, err := baseline.Producer{}.Files(baselinetest.Answers(golang.Language), o,
					&workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Length(t, files, 1, "the configurations")
				assert.Contains(t, string(files[0].Content), "\n    license: BUSL-1.1\n", "the license of the package")
			})

			t.Run("adds the NOTICE of a repository under Apache-2.0 to the archives and the packages",
				func(t *testing.T) {
					t.Parallel()
					o := goOptions()
					o.Binaries = []baseline.Command{{
						Name: "worker", Module: ".", Main: "./cmd/worker", Description: "Runs the jobs.",
						Packages: []baseline.Package{baseline.PackageDeb},
					}}
					a := baselinetest.Answers(golang.Language)
					a.License = spdx.Apache20
					files, err := baseline.Producer{}.Files(a, o, &workflow.Contribution{})
					assert.NoError(t, err, "Files")
					assert.Length(t, files, 1, "the configurations")
					expect.That(t, string(files[0].Content)).
						Contains("\n      - src: NOTICE\n        info:", "the NOTICE of the archive").
						Contains("\n      - src: NOTICE\n        dst: /usr/share/doc/worker/NOTICE\n", "the NOTICE of the package")
				})
		})
	})
}

// commands returns new commands of the cases: ergon, a command of the root module with completions,
// every package and a cask, a second command of the root module, and assertlint of the module lint
// on two platforms.
func commands() []baseline.Command {
	return []baseline.Command{
		{
			Name:        "ergon",
			Module:      ".",
			Main:        "./cmd/ergon",
			Description: "Sets up repositories and releases their packages",
			Completions: true,
			Packages:    []baseline.Package{baseline.PackageDeb, baseline.PackageRPM, baseline.PackageAPK},
			Homebrew:    true,
		},
		{
			Name:        "assertlint",
			Module:      "lint",
			Main:        "./cmd/assertlint",
			Description: `Reports the "hand-written" checks that an assertion states.`,
			Platforms:   []option.Platform{option.LinuxAMD64, option.DarwinARM64},
		},
		{
			Name:        "ergon-go-vet",
			Module:      ".",
			Main:        "./cmd/ergon-go-vet",
			Description: "Runs the analyzers of ergon.",
			Platforms:   []option.Platform{option.LinuxARM64, option.WindowsAMD64},
		},
	}
}
