// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/tool"
)

// pluginsDir is the directory beside the program of golangci-lint in the cache that keeps the
// programs of golangci-lint custom.
const pluginsDir = "plugins"

func TestToolchain(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		runs := []struct {
			name     string
			give     func(*options)
			platform option.Platform
			tool     string
			args     []string
			want     string
		}{
			{
				name: "installs a Go module with go install and runs its program",
				tool: "golangci-lint",
				args: []string{"run", "./..."},
				want: "golangci-lint run ./...\n",
			},
			{
				name: "names the program of a module of a major version after the element before it",
				give: func(o *options) { o.Tools.Lint = "example.com/tool/v2@v2.0.0" },
				tool: "golangci-lint",
				want: "tool\n",
			},
			{
				name:     "names the program of a module with .exe on Windows",
				platform: option.WindowsAMD64,
				tool:     "golangci-lint",
				want:     "golangci-lint\n",
			},
			{
				name: "installs a crate with cargo install and runs its program",
				tool: "cargo-audit",
				args: []string{"--deny", "warnings"},
				want: "cargo-audit --deny warnings\n",
			},
			{
				name:     "names the program of a crate with .exe on Windows",
				platform: option.WindowsAMD64,
				tool:     "cargo-audit",
				want:     "cargo-audit\n",
			},
			{
				name: "runs an npm package with npx",
				tool: "biome",
				args: []string{"check"},
				want: "npx --yes --package=@biomejs/biome@2.5.15 -- biome check\n",
			},
			{
				name: "runs the program of the tag program of an npm package",
				tool: "typescript",
				args: []string{"--noEmit"},
				want: "npx --yes --package=typescript@7.0.2 -- tsc --noEmit\n",
			},
		}
		for _, tt := range runs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o := demo()
				if tt.give != nil {
					tt.give(o)
				}
				r, out, _ := runner(t, map[string][]byte{})
				if tt.platform != "" {
					r.Platform = tt.platform
				}
				_, err := r.Run(t.Context(), section, o, tt.tool, tt.args)
				assert.NoError(t, err, "Run")
				assert.Equal(t, out.String(), tt.want, "the output of the tool")
			})
		}

		t.Run("runs an installed module without installing it again", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "golangci-lint", nil)
			assert.NoError(t, err, "the first Run")
			r.Env = append(r.Env, installFailsEnv+"=1")
			_, err = r.Run(t.Context(), section, demo(), "golangci-lint", []string{"again"})
			assert.NoError(t, err, "the second Run, whose go install fails")
			assert.Equal(t, out.String(), "golangci-lint\n"+"golangci-lint again\n", "the output of both runs")
		})

		t.Run("installs a module again for another version of the go command", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "golangci-lint", nil)
			assert.NoError(t, err, "the Run with the first version")
			r.Env = append(r.Env, goVersionEnv+"=go1.28.0")
			_, err = r.Run(t.Context(), section, demo(), "golangci-lint", []string{"again"})
			assert.NoError(t, err, "the Run with the second version")
			assert.Equal(t, out.String(), "golangci-lint\n"+"golangci-lint again\n", "the output of both runs")
			var programs []string
			err = filepath.WalkDir(r.Cache, func(p string, d fs.DirEntry, err error) error {
				if err == nil && d.Type().IsRegular() && strings.TrimSuffix(d.Name(), exe) == "golangci-lint" {
					programs = append(programs, p)
				}
				return err
			})
			assert.NoError(t, err, "WalkDir of the cache")
			assert.Length(t, programs, 2, "the programs of the module in the cache")
		})

		t.Run("returns ErrInstall for a go command that does not report its version", func(t *testing.T) {
			t.Parallel()
			r, _, errs := runner(t, map[string][]byte{})
			r.Env = append(r.Env, versionFailsEnv+"=1")
			_, err := r.Run(t.Context(), section, demo(), "golangci-lint", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			assert.Contains(t, errs.String(), "GOVERSION is unknown", "the output of go")
		})

		t.Run("runs an installed crate a second time", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "cargo-audit", nil)
			assert.NoError(t, err, "the first Run")
			_, err = r.Run(t.Context(), section, demo(), "cargo-audit", nil)
			assert.NoError(t, err, "the second Run")
			assert.Equal(t, strings.Count(out.String(), "cargo-audit\n"), 2, "the runs of cargo-audit")
		})

		t.Run("returns ErrInstall for a module that go install does not install", func(t *testing.T) {
			t.Parallel()
			o := demo()
			o.Tools.Lint = "example.com/broken@v1.0.0"
			r, _, errs := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, o, "golangci-lint", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			assert.Contains(t, errs.String(), "the module does not install", "the output of go")
		})

		custom := []struct {
			name     string
			give     option.Plugins
			platform option.Platform
			want     string
		}{
			{
				name: "builds golangci-lint with the plugins of its option and runs the program",
				give: option.Plugins{"assertlint": "go.dokimi.dev/assert/lint/golangci@v0.1.0"},
				want: `{"version":"v2.14.0","name":"golangci-lint","plugins":[` +
					`{"module":"go.dokimi.dev/assert/lint/golangci","version":"v0.1.0"}]}`,
			},
			{
				name: "builds each package of the plugins once in the order of the packages",
				give: option.Plugins{
					"b": "example.com/one@v1.0.0",
					"a": "example.com/two@v1.0.0",
					"c": "example.com/one@v1.0.0",
				},
				want: `{"version":"v2.14.0","name":"golangci-lint","plugins":[` +
					`{"module":"example.com/one","version":"v1.0.0"},{"module":"example.com/two","version":"v1.0.0"}]}`,
			},
			{
				name:     "names the program of golangci-lint custom with .exe on Windows",
				give:     option.Plugins{"assertlint": "go.dokimi.dev/assert/lint/golangci@v0.1.0"},
				platform: option.WindowsAMD64,
				want: `{"version":"v2.14.0","name":"golangci-lint","plugins":[` +
					`{"module":"go.dokimi.dev/assert/lint/golangci","version":"v0.1.0"}]}`,
			},
			{name: "runs the program of go install for an option without plugins", give: option.Plugins{}},
		}
		for _, tt := range custom {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, out, _ := runner(t, map[string][]byte{})
				if tt.platform != "" {
					r.Platform = tt.platform
				}
				_, err := r.Run(t.Context(), section, withPlugins(tt.give), "golangci-lint", []string{"--config"})
				assert.NoError(t, err, "Run")
				assert.Equal(t, out.String(), "golangci-lint --config\n"+tt.want+"\n", "the output of the program")
			})
		}

		t.Run("runs a program of golangci-lint custom without building it again", func(t *testing.T) {
			t.Parallel()
			o := withPlugins(option.Plugins{"assertlint": "go.dokimi.dev/assert/lint/golangci@v0.1.0"})
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, o, "golangci-lint", nil)
			assert.NoError(t, err, "the first Run")
			r.Env = append(r.Env, customFailsEnv+"=1")
			_, err = r.Run(t.Context(), section, o, "golangci-lint", []string{"again"})
			assert.NoError(t, err, "the second Run, whose golangci-lint custom fails")
			assert.Equal(t, out.String(), "golangci-lint\n"+"golangci-lint again\n", "the output of both runs")
		})

		t.Run("builds golangci-lint again for other plugins", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			first := withPlugins(option.Plugins{"lint": "example.com/lint@v1.0.0"})
			_, err := r.Run(t.Context(), section, first, "golangci-lint", nil)
			assert.NoError(t, err, "the Run with the first plugins")
			out.Reset()
			second := withPlugins(option.Plugins{"lint": "example.com/lint@v2.0.0"})
			_, err = r.Run(t.Context(), section, second, "golangci-lint", []string{"--config"})
			assert.NoError(t, err, "the Run with the second plugins")
			plugin := `{"module":"example.com/lint","version":"v2.0.0"}`
			assert.Contains(t, out.String(), plugin, "the output of the program")
		})

		t.Run("returns ErrInstall for a plugin that golangci-lint custom does not build", func(t *testing.T) {
			t.Parallel()
			r, _, errs := runner(t, map[string][]byte{})
			o := withPlugins(option.Plugins{"lint": "example.com/broken@v1.0.0"})
			_, err := r.Run(t.Context(), section, o, "golangci-lint", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			assert.Contains(t, errs.String(), "the plugin does not build", "the output of golangci-lint custom")
		})

		t.Run("returns the error of go install for a golangci-lint that does not install", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			o := withPlugins(option.Plugins{"lint": "example.com/lint@v1.0.0"})
			o.Tools.Lint = "example.com/broken@v1.0.0"
			_, err := r.Run(t.Context(), section, o, "golangci-lint", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			assert.Contains(t, err.Error(), "go install example.com/broken@v1.0.0", "the error of Run")
		})

		t.Run("returns ErrInstall for a directory of the plugins that does not create", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, withPlugins(option.Plugins{}), "golangci-lint", nil)
			assert.NoError(t, err, "the Run that installs golangci-lint")
			var programs []string
			err = filepath.WalkDir(r.Cache, func(p string, d fs.DirEntry, err error) error {
				if err == nil && d.Type().IsRegular() && strings.TrimSuffix(d.Name(), exe) == "golangci-lint" {
					programs = append(programs, p)
				}
				return err
			})
			assert.NoError(t, err, "WalkDir of the cache")
			assert.Length(t, programs, 1, "the programs of golangci-lint in the cache")
			file := filepath.Join(filepath.Dir(programs[0]), pluginsDir)
			assert.NoError(t, os.WriteFile(file, nil, 0o644), "WriteFile in place of the directory of the plugins")
			o := withPlugins(option.Plugins{"lint": "example.com/lint@v1.0.0"})
			_, err = r.Run(t.Context(), section, o, "golangci-lint", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		misnamed := []struct {
			name string
			tool string
		}{
			{name: "returns ErrInstall for a tag plugins that names a missing key", tool: "missing"},
			{name: "returns ErrInstall for a tag plugins that names an option of another type", tool: "typed"},
			{name: "returns ErrInstall for a tag plugins that names a key below an option of no group", tool: "deep"},
		}
		for _, tt := range misnamed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _, _ := runner(t, map[string][]byte{})
				_, err := r.Run(t.Context(), section, &unplugged{}, tt.tool, nil)
				assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			})
		}

		pypi := []struct {
			name string
			tool string
			args []string
			want string
		}{
			{
				name: "runs a PyPI package with uv tool run",
				tool: "ruff",
				args: []string{"check"},
				want: "uv tool run --from ruff==0.16.10 ruff check\n",
			},
			{
				name: "runs a PyPI package of the project with uv run",
				tool: "mypy",
				args: []string{"src"},
				want: "uv run --with mypy==2.4.0 -- mypy src\n",
			},
		}
		for _, tt := range pypi {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, served := demo(), map[string][]byte{}
				o.Tools.UV = uv(served, host())
				r, out, _ := runner(t, served)
				_, err := r.Run(t.Context(), section, o, tt.tool, tt.args)
				assert.NoError(t, err, "Run")
				assert.Equal(t, out.String(), tt.want, "the output of uv")
			})
		}

		t.Run("returns ErrInstall for a PyPI package of a section without uv", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, &pythonless{Tools: struct {
				Ruff option.PyPI `yaml:"ruff"`
			}{Ruff: "ruff@0.16.10"}}, "ruff", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run("returns the error of the installation of uv", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.UV = uv(served, host())
			o.Tools.UV.SHA256[host()] = digest([]byte("other"))
			r, _, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, o, "ruff", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run(
			"installs the Composer packages of the section together and runs the program with php",
			func(t *testing.T) {
				t.Parallel()
				r, out, _ := runner(t, map[string][]byte{})
				_, err := r.Run(t.Context(), section, demo(), "phpstan", []string{"analyse"})
				assert.NoError(t, err, "Run")
				program, ok := strings.CutPrefix(strings.TrimSuffix(out.String(), " analyse\n"), "php ")
				assert.True(t, ok, "the output starts with php")
				bin := filepath.Dir(program)
				files.HasContent(
					t,
					filepath.Join(bin, "..", "..", "composer.json"),
					"{\"config\": {\"allow-plugins\": true}}\n",
					"the project, which allows the plugins of its packages",
				)
				files.IsFile(t, filepath.Join(bin, "phpstan"), "the program phpstan")
				files.IsFile(
					t,
					filepath.Join(bin, "phpstan-strict-rules"),
					"the program of the package that extends it",
				)
				files.IsFile(t, filepath.Join(bin, "shim"), "the program of php-cs-fixer")
				files.Absent(t, filepath.Join(bin, "package"), "the program of the unexported field")
			},
		)

		t.Run("runs the program of the tag program of a Composer package", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "php-cs-fixer", nil)
			assert.NoError(t, err, "Run")
			assert.HasSuffix(t, out.String(), "php-cs-fixer\n", "the output of php")
		})

		t.Run("runs an installed project of Composer a second time", func(t *testing.T) {
			t.Parallel()
			r, out, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "phpstan", nil)
			assert.NoError(t, err, "the first Run")
			_, err = r.Run(t.Context(), section, demo(), "phpstan", nil)
			assert.NoError(t, err, "the second Run")
			assert.Equal(t, strings.Count(out.String(), "php "), 2, "the runs of php")
		})

		t.Run("returns ErrInstall for a project of Composer that does not create", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			r.Cache = filepath.Join(r.Cache, "file")
			assert.NoError(t, os.WriteFile(r.Cache, nil, 0o644), "WriteFile of the cache")
			_, err := r.Run(t.Context(), section, demo(), "phpstan", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run("returns ErrInstall for Composer packages that composer does not install", func(t *testing.T) {
			t.Parallel()
			o := demo()
			o.Tools.Strict = "broken/rules@1.0.0"
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, o, "phpstan", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})
	})
}

// pythonless are options with a PyPI package and without the uv that runs it.
type pythonless struct {
	Tools struct {
		Ruff option.PyPI `yaml:"ruff"`
	} `yaml:"tools"`
}

// Validate returns nil.
func (*pythonless) Validate() error {
	return nil
}

// plugged are options whose golangci-lint builds with the module plugins of lint.plugins.
type plugged struct {
	Tools struct {
		Lint option.Module `yaml:"golangci-lint" plugins:"lint.plugins"`
	} `yaml:"tools"`
	Lint struct {
		Plugins option.Plugins `yaml:"plugins"`
	} `yaml:"lint"`
}

// Validate returns nil.
func (*plugged) Validate() error {
	return nil
}

// unplugged are options whose tags plugins name no option of plugins: a key that the options do not
// have, an option of another type, and a key below an option that is no group.
type unplugged struct {
	Tools struct {
		Missing option.Module `yaml:"missing" plugins:"lint.missing"`
		Typed   option.Module `yaml:"typed"   plugins:"tools.typed"`
		Deep    option.Module `yaml:"deep"    plugins:"tools.deep.plugins"`
	} `yaml:"tools"`
	Lint struct{} `yaml:"lint"`
}

// Validate returns nil.
func (*unplugged) Validate() error {
	return nil
}

// withPlugins returns options of golangci-lint v2.14.0 with the module plugins p.
func withPlugins(p option.Plugins) *plugged {
	o := &plugged{}
	o.Tools.Lint = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0"
	o.Lint.Plugins = p
	return o
}
