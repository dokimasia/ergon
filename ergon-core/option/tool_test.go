// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestTool(t *testing.T) {
	t.Parallel()

	t.Run("Module", func(t *testing.T) {
		t.Parallel()
		m := option.Module("golang.org/x/vuln/cmd/govulncheck@v1.8.0")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the path of the module", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, m.Package(), "golang.org/x/vuln/cmd/govulncheck", "Package")
			})

			t.Run("returns the whole text of a module without a version", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, option.Module("golang.org/x/vuln").Package(), "golang.org/x/vuln", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, m.Version(), "v1.8.0", "Version")
			})

			t.Run("returns the empty version of a module without one", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, option.Module("golang.org/x/vuln").Version(), "", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Module
			}{
				{name: "returns nil for a module at a release", give: m},
				{
					name: "returns nil for a module at a pseudo-version",
					give: "golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68",
				},
				{
					name: "returns nil for a module of a vanity domain",
					give: "go.dokimi.dev/ergon/lang/go/cmd/ergon-go-vet@v1.0.0",
				},
				{
					name: "returns nil for a module of a major version",
					give: "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0",
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Module
			}{
				{name: "returns ErrInvalid for the empty module", give: ""},
				{name: "returns ErrInvalid for a module without a version", give: "golang.org/x/vuln/cmd/govulncheck"},
				{name: "returns ErrInvalid for a version without a v", give: "golang.org/x/vuln/cmd/govulncheck@1.8.0"},
				{name: "returns ErrInvalid for a path without a domain", give: "cmd/govulncheck@v1.8.0"},
				{name: "returns ErrInvalid for a domain without a path", give: "golang.org@v1.8.0"},
				{name: "returns ErrInvalid for a path with a space", give: "golang.org/x/vuln cmd@v1.8.0"},
				{name: "returns ErrInvalid for a version with a dollar sign", give: "golang.org/x/vuln@v$(V)"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("PyPI", func(t *testing.T) {
		t.Parallel()
		p := option.PyPI("pip-audit@2.10.1")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the name of the package", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, p.Package(), "pip-audit", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, p.Version(), "2.10.1", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.PyPI
			}{
				{name: "returns nil for a package with a hyphen", give: p},
				{name: "returns nil for a package with a dot", give: "zope.interface@7.0"},
				{name: "returns nil for a version of a release candidate", give: "mypy@2.4.0rc1"},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.PyPI
			}{
				{name: "returns ErrInvalid for a package without a version", give: "ruff"},
				{name: "returns ErrInvalid for an empty package", give: "@0.16.10"},
				{name: "returns ErrInvalid for a package that starts with a hyphen", give: "-ruff@0.16.10"},
				{name: "returns ErrInvalid for a version with a number sign", give: "ruff@0.16#1"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("NPM", func(t *testing.T) {
		t.Parallel()
		n := option.NPM("@biomejs/biome@2.5.15")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the package of a scope with its own @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, n.Package(), "@biomejs/biome", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, n.Version(), "2.5.15", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.NPM
			}{
				{name: "returns nil for a package of a scope", give: n},
				{name: "returns nil for a package without a scope", give: "typescript@7.0.2"},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.NPM
			}{
				{name: "returns ErrInvalid for a package in uppercase", give: "TypeScript@7.0.2"},
				{name: "returns ErrInvalid for a scope without a package", give: "@biomejs@2.5.15"},
				{name: "returns ErrInvalid for a package without a version", give: "@biomejs/biome"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Crate", func(t *testing.T) {
		t.Parallel()
		c := option.Crate("cargo-audit@0.22.2")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the name of the crate", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.Package(), "cargo-audit", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.Version(), "0.22.2", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for a crate and a version", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, c.Validate(), "Validate")
			})

			invalid := []struct {
				name string
				give option.Crate
			}{
				{name: "returns ErrInvalid for a crate that starts with a digit", give: "9audit@0.22.2"},
				{name: "returns ErrInvalid for a crate without a version", give: "cargo-audit"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Maven", func(t *testing.T) {
		t.Parallel()
		m := option.Maven("com.pinterest.ktlint:ktlint-cli@1.8.0")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the group and the artifact", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, m.Package(), "com.pinterest.ktlint:ktlint-cli", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, m.Version(), "1.8.0", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Maven
			}{
				{name: "returns nil for an artifact with a hyphen", give: m},
				{name: "returns nil for a group of three parts", give: "net.sourceforge.pmd:pmd-java@7.28.0"},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Maven
			}{
				{name: "returns ErrInvalid for a group without an artifact", give: "com.pinterest.ktlint@1.8.0"},
				{name: "returns ErrInvalid for coordinates of three parts", give: "com.pinterest:ktlint:cli@1.8.0"},
				{name: "returns ErrInvalid for an artifact without a version", give: "com.pinterest.ktlint:ktlint-cli"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Composer", func(t *testing.T) {
		t.Parallel()
		c := option.Composer("php-cs-fixer/shim@3.95.27")

		t.Run("Package", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the vendor and the package", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.Package(), "php-cs-fixer/shim", "Package")
			})
		})

		t.Run("Version", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the version after the last @", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.Version(), "3.95.27", "Version")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Composer
			}{
				{name: "returns nil for a vendor with a hyphen", give: c},
				{name: "returns nil for a package with hyphens", give: "phpstan/phpstan-strict-rules@2.0.12"},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Composer
			}{
				{name: "returns ErrInvalid for a package without a vendor", give: "phpstan@2.2.17"},
				{name: "returns ErrInvalid for a vendor in uppercase", give: "PHPStan/phpstan@2.2.17"},
				{name: "returns ErrInvalid for a package without a version", give: "phpstan/phpstan"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})
}
