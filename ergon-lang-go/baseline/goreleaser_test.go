// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"path"
	"testing"

	"go.dokimi.dev/assert"
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
			}
			for _, tt := range splits {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					o := goOptions()
					o.Binaries = []baseline.Command{{
						Name: "worker", Module: ".", Main: "./cmd/worker", Description: "Runs the jobs.",
						Platforms: tt.platforms,
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
		})
	})
}

// commands returns new commands of the cases: ergon, a cobra program of the root module with every
// package and a cask, a second command of the root module, and assertlint of the module lint on two
// platforms.
func commands() []baseline.Command {
	return []baseline.Command{
		{
			Name:        "ergon",
			Module:      ".",
			Main:        "./cmd/ergon",
			Description: "Sets up repositories and releases their packages.",
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
