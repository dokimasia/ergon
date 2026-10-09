// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/lang/go/baseline"
)

func TestCommand(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give func(*baseline.Command)
		}{
			{name: "returns nil for a command of the root module", give: func(*baseline.Command) {}},
			{name: "returns nil for a command of a module in a directory", give: func(c *baseline.Command) {
				c.Module = "tools/lint"
			}},
			{name: "returns nil for a command with its own license", give: func(c *baseline.Command) {
				c.License = spdx.ID("Apache-2.0")
			}},
			{name: "returns nil for a cask of a command of Linux alone", give: func(c *baseline.Command) {
				c.Platforms, c.Packages = []option.Platform{option.LinuxARM64}, nil
			}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := ergonCommand()
				tt.give(&c)
				assert.NoError(t, c.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give func(*baseline.Command)
			want string
		}{
			{
				name: "returns ErrInvalid for a name with an uppercase letter",
				give: func(c *baseline.Command) { c.Name = "Ergon" },
				want: `the command "Ergon", whose name is not lowercase letters, digits and -`,
			},
			{
				name: "returns ErrInvalid for a module outside the repository",
				give: func(c *baseline.Command) { c.Module = "../other" },
				want: `the command ergon has the module "../other", which is no clean relative directory`,
			},
			{
				name: "returns ErrInvalid for a module that is not clean",
				give: func(c *baseline.Command) { c.Module = "lint/" },
				want: `the command ergon has the module "lint/", which is no clean relative directory`,
			},
			{
				name: "returns ErrInvalid for a module with a space",
				give: func(c *baseline.Command) { c.Module = "my lint" },
				want: `the command ergon has the module "my lint", which is no clean relative directory`,
			},
			{
				name: "returns ErrInvalid for a package without ./",
				give: func(c *baseline.Command) { c.Main = "cmd/ergon" },
				want: `the command ergon has the package "cmd/ergon", which is no clean package under ./`,
			},
			{
				name: "returns ErrInvalid for a package outside the module",
				give: func(c *baseline.Command) { c.Main = "./../cmd/ergon" },
				want: `the command ergon has the package "./../cmd/ergon", which is no clean package under ./`,
			},
			{
				name: "returns ErrInvalid for the package ./",
				give: func(c *baseline.Command) { c.Main = "./" },
				want: `the command ergon has the package "./", which is no clean package under ./`,
			},
			{
				name: "returns ErrInvalid for a description that spans lines",
				give: func(c *baseline.Command) { c.Description = "One.\nTwo." },
				want: "the command ergon has a description that is empty or spans lines",
			},
			{
				name: "returns ErrInvalid for an empty description",
				give: func(c *baseline.Command) { c.Description = " " },
				want: "the command ergon has a description that is empty or spans lines",
			},
			{
				name: "returns ErrInvalid for a license that ergon does not know",
				give: func(c *baseline.Command) { c.License = spdx.ID("WTFPL-2") },
				want: `the command ergon has the license "WTFPL-2", which is no identifier of --license`,
			},
			{
				name: "returns ErrInvalid for a platform that is not valid",
				give: func(c *baseline.Command) { c.Platforms = []option.Platform{"linux/386"} },
				want: `the command ergon: option: invalid value: platform "linux/386"`,
			},
			{
				name: "returns ErrInvalid for a platform named twice",
				give: func(c *baseline.Command) {
					c.Platforms = []option.Platform{option.LinuxAMD64, option.LinuxAMD64}
				},
				want: "the command ergon names the platform linux/amd64 twice",
			},
			{
				name: "returns ErrInvalid for a package format that ergon does not build",
				give: func(c *baseline.Command) { c.Packages = []baseline.Package{"msix"} },
				want: `the command ergon has the package "msix", which is not deb, rpm or apk, or is named twice`,
			},
			{
				name: "returns ErrInvalid for a package format named twice",
				give: func(c *baseline.Command) {
					c.Packages = []baseline.Package{baseline.PackageDeb, baseline.PackageDeb}
				},
				want: `the command ergon has the package "deb", which is not deb, rpm or apk, or is named twice`,
			},
			{
				name: "returns ErrInvalid for packages without a Linux platform",
				give: func(c *baseline.Command) {
					c.Platforms, c.Homebrew = []option.Platform{option.WindowsAMD64}, false
				},
				want: "the command ergon has packages of Linux and no Linux platform",
			},
			{
				name: "returns ErrInvalid for a cask without a darwin or Linux platform",
				give: func(c *baseline.Command) {
					c.Platforms, c.Packages = []option.Platform{option.WindowsARM64}, nil
				},
				want: "the command ergon has a cask and no darwin or Linux platform",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := ergonCommand()
				tt.give(&c)
				err := c.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})
}

// ergonCommand returns a new valid command of the cases: ergon of the root module, a cobra program
// with every package and a cask, on every platform.
func ergonCommand() baseline.Command {
	return baseline.Command{
		Name:        "ergon",
		Module:      ".",
		Main:        "./cmd/ergon",
		Description: "Sets up repositories and releases their packages.",
		Completions: true,
		Packages:    []baseline.Package{baseline.PackageDeb, baseline.PackageRPM, baseline.PackageAPK},
		Homebrew:    true,
	}
}
