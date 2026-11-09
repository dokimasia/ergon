// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// setupGo is the pin of setup-go at the baseline.
var setupGo = goOptions().CI.Actions.SetupGo

// margin is the number of minutes of a nightly job of mutate or longtest that the run of make does
// not get, which the job keeps for its setup and its cache.
const margin = 10

// lintCache is the cache of golangci-lint of the job check-go at the baseline.
var lintCache = workflow.Cache{
	ID:    "golangci-lint",
	Name:  "the cache of golangci-lint",
	Path:  "${{ runner.os == 'Windows' && '~/AppData/Local/golangci-lint' || runner.os == 'macOS' && '~/Library/Caches/golangci-lint' || '~/.cache/golangci-lint' }}",
	Files: []string{"**/go.sum", "go.work", ".golangci.yml", ".ergon.yaml", ".ergon/init.lock"},
	Days:  7,
}

// historyCache is the cache of the history of dokimi-mutate-go of the job mutate-go.
var historyCache = workflow.Cache{
	ID:    "dokimi-mutate-go-history",
	Name:  "the history of dokimi-mutate-go",
	Path:  "${{ runner.temp }}/dokimi-mutate-go-history",
	Input: "history",
}

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

// requireWork is the step of the job of Go that fails a repository without go.work.
var requireWork = workflow.Step{
	Name: "Require go.work",
	If:   "hashFiles('go.work') == ''",
	Run: []string{
		`echo "::error::The targets of Go run in the modules of go.work, which the repository lacks."`,
		"exit 1",
	},
}

// ownPackages is the step before the dependency review that appends the package URL of each
// directory replace of a module of go.work to ERGON_OWN_PACKAGES.
var ownPackages = workflow.Step{
	Name: "Leave the modules of go.work out of the license check",
	If:   "hashFiles('go.work') != ''",
	Run: []string{
		`own="$(go list -m -f '{{.Path}}' | jq -Rn '[inputs]')"`,
		`purls="$(go list -m -f '{{.GoMod}}' |`,
		`  while IFS= read -r file; do go mod edit -json "$file"; done |`,
		`  jq -r --argjson own "$own" '.Replace[]? | select(.New.Version == null)`,
		`    | select(.Old.Path | IN($own[])) | "pkg:golang/" + (.New.Path | sub("/+$"; ""))' |`,
		`  sort -u | paste -sd , -)"`,
		`if [ -n "$purls" ]; then`,
		`  echo "ERGON_OWN_PACKAGES=${ERGON_OWN_PACKAGES:+$ERGON_OWN_PACKAGES,}$purls" >> "$GITHUB_ENV"`,
		`fi`,
	},
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Tools", func(t *testing.T) {
		t.Parallel()

		t.Run("names a group of linters in the tag plugins of golangci-lint", func(t *testing.T) {
			t.Parallel()
			lint, ok := reflect.TypeFor[baseline.Tools]().FieldByName("GolangCILint")
			assert.True(t, ok, "Tools has the field GolangCILint")
			fields, err := options.Fields(goOptions())
			assert.NoError(t, err, "Fields of the options")
			types := map[string]reflect.Type{}
			for _, f := range fields {
				types[f.Key] = f.Type
			}
			key := lint.Tag.Get(option.PluginsTag)
			assert.NotNil(t, types[key], "the type of the option "+key)
			assert.True(t, types[key].Implements(reflect.TypeFor[option.Linters]()),
				"the option "+key+" implements Linters")
		})
	})

	t.Run("Lint", func(t *testing.T) {
		t.Parallel()

		t.Run("Linters", func(t *testing.T) {
			t.Parallel()

			analyzers := option.Module("go.dokimi.dev/lint/golangci@v0.2.0")
			assertlint := option.Module("go.dokimi.dev/assert/lint/golangci@v0.1.0")
			tests := []struct {
				name string
				give option.Plugins
				want option.Plugins
			}{
				{
					name: "returns errorprefix and skipexpiry of the analyzers",
					give: option.Plugins{},
					want: option.Plugins{"errorprefix": analyzers, "skipexpiry": analyzers},
				},
				{
					name: "returns each plugin beside the analyzers",
					give: option.Plugins{"assertlint": assertlint},
					want: option.Plugins{"errorprefix": analyzers, "skipexpiry": analyzers, "assertlint": assertlint},
				},
				{
					name: "returns a plugin of the name of an analyzer in place of the analyzer",
					give: option.Plugins{"errorprefix": assertlint},
					want: option.Plugins{"errorprefix": assertlint, "skipexpiry": analyzers},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					l := baseline.Lint{Analyzers: analyzers, Plugins: tt.give}
					assert.Equal(t, l.Linters(), tt.want, "the linters")
				})
			}

			t.Run("returns a map that shares nothing with the plugins", func(t *testing.T) {
				t.Parallel()
				l := baseline.Lint{Analyzers: analyzers, Plugins: option.Plugins{"assertlint": assertlint}}
				assert.Pure(t, func() option.Plugins { return l.Plugins }, func() {
					l.Linters()["otherlint"] = "example.com/otherlint@v1.0.0"
				}, "the plugins")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			for _, give := range []int{0, 7} {
				t.Run(
					"returns nil for a cache invalidation interval of "+strconv.Itoa(give)+" days",
					func(t *testing.T) {
						t.Parallel()
						assert.NoError(t, baseline.Lint{CacheInvalidationInterval: give}.Validate(), "Validate")
					},
				)
			}

			t.Run("returns ErrInvalid for a cache invalidation interval below 0", func(t *testing.T) {
				t.Parallel()
				err := baseline.Lint{CacheInvalidationInterval: -1}.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				want := "option: invalid value: lint.cache-invalidation-interval -1, which is less than 0"
				assert.Equal(t, err.Error(), want, "the error")
			})
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns nil for every step of Go", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{
				option.StepLint, option.StepTest, option.StepRace, option.StepLongTest, option.StepFuzz,
				option.StepBench, option.StepMutate, option.StepGenerate, option.StepAudit,
			}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns nil for every nightly step of Go", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Nightly = option.Nightly{
				option.StepLongTest: margin + 1, option.StepFuzz: 1, option.StepBench: 1, option.StepMutate: margin + 1,
			}
			assert.NoError(t, o.Validate(), "Validate")
		})

		for _, step := range []option.Step{option.StepMutate, option.StepLongTest} {
			t.Run("returns ErrInvalid for a nightly limit of "+strconv.Itoa(margin)+" minutes for "+string(step),
				func(t *testing.T) {
					t.Parallel()
					o := goOptions()
					o.Nightly = option.Nightly{step: margin}
					err := o.Validate()
					assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
					assert.Contains(t, err.Error(), "nightly gives "+string(step)+" "+strconv.Itoa(margin)+" minutes",
						"the error")
				})
		}

		t.Run("returns ErrInvalid for the step fmt, which formats the sources", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command line", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate = option.Generate{}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns ErrInvalid for a nightly step other than longtest, fuzz, bench and mutate", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Nightly = option.Nightly{option.StepFuzz: 120, option.StepRace: 30}
			err := o.Validate()
			assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
			assert.Contains(t, err.Error(), "nightly names race", "the error")
		})

		t.Run("returns nil for commands with a cask and a tap", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Binaries = commands()
			o.Homebrew.Tap = tap
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns nil for a provider beside a command of another module", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Binaries = []baseline.Command{commands()[1], providerCommand()}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns nil for a command named after the macOS build of a command without cgo", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Binaries = commands()
			o.Binaries[1].Name = "ergon-darwin"
			o.Homebrew.Tap = tap
			assert.NoError(t, o.Validate(), "Validate")
		})

		binaries := []struct {
			name string
			give func(*baseline.Options)
			want string
		}{
			{
				name: "returns ErrInvalid for a command that is not valid",
				give: func(o *baseline.Options) { o.Binaries[0].Main = "cmd/ergon" },
				want: `the command ergon has the package "cmd/ergon"`,
			},
			{
				name: "returns ErrInvalid for two commands of one name",
				give: func(o *baseline.Options) { o.Binaries[1].Name = "ergon" },
				want: "binaries names the command ergon twice",
			},
			{
				name: "returns ErrInvalid for a command with a cask without a tap",
				give: func(o *baseline.Options) { o.Homebrew.Tap = "" },
				want: "the command ergon has a cask, and homebrew.tap is empty",
			},
			{
				name: "returns ErrInvalid for a command named after the UPX build of another",
				give: func(o *baseline.Options) { o.Binaries[1].Name = "ergon-upx" },
				want: "the command ergon-upx has the name of the UPX build of the command ergon",
			},
			{
				name: "returns ErrInvalid for a command named after the macOS build of a command with cgo",
				give: func(o *baseline.Options) {
					o.Binaries[0].CGO = true
					o.Binaries[1].Name = "ergon-darwin"
				},
				want: "the command ergon-darwin has the name of the macOS build of the command ergon",
			},
			{
				name: "returns ErrInvalid for a provider beside another command of its module",
				give: func(o *baseline.Options) { o.Binaries = append(o.Binaries, providerCommand()) },
				want: "the provider terraform-provider-demo has another command in its module",
			},
		}
		for _, tt := range binaries {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.Binaries = commands()
				o.Homebrew.Tap = tap
				tt.give(o)
				err := o.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})

	t.Run("Homebrew", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			for _, give := range []string{"", tap} {
				t.Run("returns nil for the tap "+give, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, baseline.Homebrew{Tap: give}.Validate(), "Validate")
				})
			}

			t.Run("returns ErrInvalid for a tap that is not owner/name", func(t *testing.T) {
				t.Parallel()
				err := baseline.Homebrew{Tap: "homebrew-tap"}.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				want := `option: invalid value: homebrew.tap "homebrew-tap", which is not owner/name`
				assert.Equal(t, err.Error(), want, "the error")
			})
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run(
			"returns the job, the release steps, the analysis and the updates of Go at the baseline",
			func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, goOptions().Contribution(), workflow.Contribution{
					Jobs: []workflow.Job{{
						ID:          "check-go",
						Name:        "Go",
						Permissions: map[string]string{"contents": "read"},
						Setup: &workflow.Setup{
							Files:        "**/go.mod",
							VersionFiles: "go.work",
							Runners:      option.Runners{},
							Versions:     []string{},
							Steps: []workflow.Step{
								requireWork,
								{
									Name: "Set up Go",
									Uses: setupGo,
									With: map[string]string{
										"go-version-file":       "go.work",
										"cache-dependency-path": "**/go.sum",
									},
								},
							},
							Timeout: 30,
						},
						Tools:  true,
						Caches: []workflow.Cache{lintCache},
						Steps:  []workflow.Step{{Name: "Check Go", Run: []string{"make --keep-going check-go"}}},
					}},
					Nightly: []workflow.Job{
						nightlyJob("fuzz-go", "Fuzz Go", 120),
						nightlyJob("bench-go", "Bench Go", 45),
						mutateJob(60),
					},
					Release: []workflow.Step{{
						Name: "Set up Go",
						If:   "hashFiles('go.work') != ''",
						Uses: setupGo,
						With: map[string]string{"go-version-file": "go.work", "cache-dependency-path": "**/go.sum"},
					}},
					CodeQL: []workflow.CodeQL{{
						Language:  "go",
						Name:      "Go",
						BuildMode: "autobuild",
						Files:     "go.work",
						Steps: []workflow.Step{
							{
								Name: "Set up Go",
								Uses: setupGo,
								With: map[string]string{
									"go-version-file":       "go.work",
									"cache-dependency-path": "**/go.sum",
								},
							},
							{
								Name: "Fetch the modules of go.work from their origin",
								Run: []string{
									`echo "GOPRIVATE=$(go list -m -f '{{.Path}}' | paste -sd , -)" >> "$GITHUB_ENV"`,
								},
							},
						},
						Timeout: 30,
					}},
					Review: []workflow.Step{
						{
							Name: "Set up Go",
							If:   "hashFiles('go.work') != ''",
							Uses: setupGo,
							With: map[string]string{"go-version-file": "go.work", "cache": "false"},
						},
						ownPackages,
					},
					Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/", "/**/*"}}},
				}, "the contribution")
			},
		)

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := goOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup of every job", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			got := o.Contribution()
			assert.Length(t, got.Nightly, 3, "the nightly jobs")
			want := []workflow.Step{
				requireWork,
				{
					Name: "Set up Go",
					Uses: setupGo,
					With: map[string]string{"go-version-file": "go.work", "cache-dependency-path": "**/go.sum"},
				},
				installTypeScript,
			}
			for _, j := range slices.Concat(got.Jobs, got.Nightly) {
				expect.Equal(t, j.Setup.Steps, want, "the steps of the setup of "+j.ID)
			}
		})

		t.Run("sets up the version of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Runners = option.Runners{"ubuntu-26.04"}
			o.CI.Versions = []string{"1.27", "1.26"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Runners, []string{"ubuntu-26.04"}, "the runners")
			assert.Equal(t, setup.Versions, []string{"1.27", "1.26"}, "the versions")
			assert.Equal(t, setup.Steps[1].With, map[string]string{
				"go-version":            "${{ matrix.version }}",
				"cache-dependency-path": "**/go.sum",
			}, "the inputs of setup-go")
		})

		t.Run("sets up the version of go.work before the CodeQL analysis for versions of the options",
			func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.CI.Versions = []string{"1.27", "1.26"}
				got := o.Contribution().CodeQL
				assert.Length(t, got, 1, "the analyses")
				assert.Equal(t, got[0].Steps[0].With, map[string]string{
					"go-version-file":       "go.work",
					"cache-dependency-path": "**/go.sum",
				}, "the inputs of setup-go")
			})

		t.Run("returns a review step that writes pkg:golang/.. for a replace of ../", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("the step runs in the job dependency-review of security.yml on the Linux runner")
			}
			review := goOptions().Contribution().Review
			assert.Length(t, review, 2, "the steps before the dependency review")
			dir := files.Workspace(t, files.Tree{
				"go.work": files.Text("go 1.27\n\nuse (\n\t.\n\t./examples\n)\n"),
				"go.mod":  files.Text("module example.com/root\n\ngo 1.27\n"),
				"examples/go.mod": files.Text("module example.com/examples\n\ngo 1.27\n\n" +
					"require example.com/root v0.0.0\n\nreplace example.com/root => ../\n"),
				"env": files.Text(""),
			})
			env := filepath.Join(dir, "env")
			// bash -e is the shell of a step of GitHub Actions without a shell on Linux.
			cmd := exec.CommandContext(t.Context(), "bash", "-e", "-c", strings.Join(review[1].Run, "\n"))
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=auto", "GITHUB_ENV="+env)
			out, err := cmd.CombinedOutput()
			assert.NoError(t, err, "the step:\n"+string(out))
			files.HasContent(t, env, workflow.OwnPackages+"=pkg:golang/..\n", "the file of GITHUB_ENV")
		})

		t.Run("sets up the version of go.work before the dependency review for versions of the options",
			func(t *testing.T) {
				t.Parallel()
				o := goOptions()
				o.CI.Versions = []string{"1.27", "1.26"}
				got := o.Contribution().Review
				assert.Length(t, got, 2, "the steps before the dependency review")
				assert.Equal(t, got[0].With, map[string]string{"go-version-file": "go.work", "cache": "false"},
					"the inputs of setup-go")
			})

		t.Run("sets up the version of go.work in each nightly job for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Runners = option.Runners{"macos-26"}
			o.CI.Versions = []string{"1.27", "1.26"}
			got := o.Contribution().Nightly
			assert.Length(t, got, 3, "the nightly jobs")
			for _, j := range got {
				expect.Equal(t, j.Setup, nightlyJob(j.ID, j.Name, j.Setup.Timeout).Setup, "the setup of "+j.ID)
			}
		})

		t.Run("sets GO_MUTATE_TIMEOUT of the job mutate-go to its limit minus the margin", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Nightly = option.Nightly{option.StepMutate: 120}
			assert.Equal(t, o.Contribution().Nightly, []workflow.Job{mutateJob(120)}, "the nightly jobs")
		})

		t.Run("runs make longtest-go with GO_LONGTEST_TIMEOUT of its limit minus the margin", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Nightly = option.Nightly{option.StepLongTest: 60}
			want := nightlyJob("longtest-go", "Longtest Go", 60)
			want.Steps = []workflow.Step{{
				Name: "Longtest Go",
				Run:  []string{"make longtest-go GO_LONGTEST_TIMEOUT=" + strconv.Itoa(60-margin) + "m"},
			}}
			assert.Equal(t, o.Contribution().Nightly, []workflow.Job{want}, "the nightly jobs")
		})

		t.Run("keeps no cache of golangci-lint for a cache invalidation interval of 0", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Lint.CacheInvalidationInterval = 0
			got := o.Contribution().Jobs
			assert.Length(t, got, 1, "the jobs")
			assert.Empty(t, got[0].Caches, "the caches of check-go")
		})

		t.Run("keeps no cache of golangci-lint for a check without lint", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{option.StepTest}
			got := o.Contribution().Jobs
			assert.Length(t, got, 1, "the jobs")
			assert.Empty(t, got[0].Caches, "the caches of check-go")
		})

		t.Run("returns the assets of the commands with the tap of a command with a cask", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Binaries = commands()
			o.Homebrew.Tap = tap
			assert.Equal(t, o.Contribution().Assets, &workflow.Assets{Tap: tap}, "the assets")
		})

		t.Run("returns assets without a tap for commands without a cask", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Binaries = commands()[1:]
			o.Homebrew.Tap = tap
			assert.Equal(t, o.Contribution().Assets, &workflow.Assets{}, "the assets")
		})

		t.Run("returns no nightly job for a nightly without steps", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Nightly = option.Nightly{}
			assert.Empty(t, o.Contribution().Nightly, "the nightly jobs")
		})

		t.Run("sets up the version of go.work in a release for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Versions = []string{"1.27", "1.26"}
			release := o.Contribution().Release
			assert.Length(t, release, 1, "the release steps")
			assert.Equal(t, release[0].With, map[string]string{
				"go-version-file":       "go.work",
				"cache-dependency-path": "**/go.sum",
			}, "the inputs of setup-go")
		})
	})
}

// nightlyJob returns the nightly job id of Go named name, which runs make id on the version of
// go.work, with a limit of minutes.
func nightlyJob(id, name string, minutes int) workflow.Job {
	return workflow.Job{
		ID:          id,
		Name:        name,
		Permissions: map[string]string{"contents": "read"},
		Setup: &workflow.Setup{
			Files:        "**/go.mod",
			VersionFiles: "go.work",
			Steps: []workflow.Step{
				requireWork,
				{
					Name: "Set up Go",
					Uses: setupGo,
					With: map[string]string{"go-version-file": "go.work", "cache-dependency-path": "**/go.sum"},
				},
			},
			Timeout: minutes,
		},
		Tools: true,
		Steps: []workflow.Step{{Name: name, Run: []string{"make " + id}}},
	}
}

// mutateJob returns the nightly job mutate-go with a limit of minutes, which keeps the history of
// dokimi-mutate-go and ends the run of make the margin before its limit.
func mutateJob(minutes int) workflow.Job {
	j := nightlyJob("mutate-go", "Mutate Go", minutes)
	j.Caches = []workflow.Cache{historyCache}
	j.Steps = []workflow.Step{{
		Name: "Mutate Go",
		Env:  map[string]string{"DOKIMI_MUTATE_GO_HISTORY": historyCache.Path},
		Run:  []string{"make mutate-go GO_MUTATE_TIMEOUT=" + strconv.Itoa(minutes-margin) + "m"},
	}}
	return j
}

// goOptions returns new options of the section go at the baseline.
func goOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
