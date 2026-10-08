// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package rewrite_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/internal/rewrite"
	"go.dokimi.dev/ergon/service/pin"
)

// source is the directory of the package of producers of the cases, whose literals state the
// earlier releases of the pins of the cases.
var source = filepath.Join("testdata", "source")

// The files of the package of the cases.
const (
	producerFile = "producer.go"
	othersFile   = "others.go"
	varsFile     = "vars.go"
)

// dirMark marks the place of the directory of a case in the text of an error.
const dirMark = "<dir>"

// The digests and the commits of the cases: 64 and 40 copies of one character.
var (
	uvLinux      = strings.Repeat("1", 64)
	uvDarwin     = strings.Repeat("2", 64)
	checkLinux   = strings.Repeat("3", 64)
	nextLinux    = strings.Repeat("a", 64)
	nextDarwin   = strings.Repeat("b", 64)
	nextCheck    = strings.Repeat("c", 64)
	otherDigest  = strings.Repeat("d", 64)
	checkout     = strings.Repeat("1", 40)
	nextCheckout = strings.Repeat("2", 40)
)

// The pins of the producer Producer of the cases.
var (
	lintPin = pin.Pin{
		Value:   option.Module("example.com/lint/cmd/lint@v1.2.0"),
		Key:     "producer.tools.lint",
		Version: "v1.2.0",
		Field:   []string{"Tools", "Lint"},
		Kind:    pin.KindModule,
	}
	uvPin = pin.Pin{
		Value: option.UV{Binary: option.Binary{
			SHA256:  map[option.Platform]string{option.LinuxAMD64: uvLinux, option.DarwinARM64: uvDarwin},
			Version: "0.10.0",
		}},
		Key:     "producer.tools.uv",
		Version: "0.10.0",
		Field:   []string{"Tools", "UV"},
		Kind:    pin.KindBinary,
	}
	checkPin = pin.Pin{
		Value: option.UV{Binary: option.Binary{
			SHA256:  map[option.Platform]string{option.LinuxAMD64: checkLinux, option.LinuxARM64: checkLinux},
			Version: "1.0.0",
		}},
		Key:     "producer.tools.check",
		Version: "1.0.0",
		Field:   []string{"Tools", "Check"},
		Kind:    pin.KindBinary,
	}
	checkoutPin = pin.Pin{
		Value:   workflow.Action{Uses: "actions/checkout", Commit: checkout, Release: "v7.0.1"},
		Key:     "producer.ci.actions.checkout",
		Version: "v7.0.1",
		Field:   []string{"CI", "Actions", "Checkout"},
		Kind:    pin.KindAction,
	}
)

// kinds are an update of a pin of each kind of the producer Producer of the cases, in its two
// files.
var kinds = []rewrite.Update{
	{Pin: lintPin, Release: pin.Release{Version: "v1.10.0"}},
	{
		Pin: pin.Pin{
			Value:   option.PyPI("audit@2.0.0"),
			Key:     "producer.tools.audit",
			Version: "2.0.0",
			Field:   []string{"Tools", "Audit"},
			Kind:    pin.KindPyPI,
		},
		Release: pin.Release{Version: "2.1.0"},
	},
	{
		Pin: uvPin,
		Release: pin.Release{
			Digests: map[option.Platform]string{option.LinuxAMD64: nextLinux, option.DarwinARM64: nextDarwin},
			Version: "0.11.0",
		},
	},
	{
		Pin: checkPin,
		Release: pin.Release{
			Digests: map[option.Platform]string{option.LinuxAMD64: nextCheck, option.LinuxARM64: nextCheck},
			Version: "1.1.0",
		},
	},
	{
		Pin: pin.Pin{
			Value:   option.Version("v6.0.0"),
			Key:     "producer.hooks",
			Version: "v6.0.0",
			Field:   []string{"Hooks"},
			Kind:    pin.KindGitHub,
		},
		Release: pin.Release{Version: "v6.1.0"},
	},
	{
		Pin: pin.Pin{
			Value:   option.NPM("format@1.0.0"),
			Key:     "producer.extra.format",
			Version: "1.0.0",
			Field:   []string{"Extra", "Format"},
			Kind:    pin.KindNPM,
		},
		Release: pin.Release{Version: "1.1.0"},
	},
	{
		Pin: pin.Pin{
			Value:   option.Crate("second@1.0.0"),
			Key:     "producer.second.format",
			Version: "1.0.0",
			Field:   []string{"Second", "Format"},
			Kind:    pin.KindCrate,
		},
		Release: pin.Release{Version: "1.2.0"},
	},
	{Pin: checkoutPin, Release: pin.Release{Version: "v7.1.0", Commit: nextCheckout}},
}

func TestRewrite(t *testing.T) {
	t.Parallel()

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("rewrites the literal of each kind of pin", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			written, err := rewrite.Apply(dir, "Producer", kinds)
			assert.NoError(t, err, "Apply")
			expect.Equal(
				t,
				written,
				[]string{filepath.Join(dir, producerFile), filepath.Join(dir, varsFile)},
				"the files that Apply wrote",
			)
			golden.MatchTree(t, "kinds", os.DirFS(dir), golden.ShouldUpdate())
		})

		t.Run("rewrites the Options method of a pointer receiver", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			update := rewrite.Update{Pin: lintPin, Release: pin.Release{Version: "v1.1.0"}}
			update.Pin.Value = option.Module("example.com/lint/cmd/lint@v1.0.0")
			update.Pin.Version = "v1.0.0"
			written, err := rewrite.Apply(dir, "Other", []rewrite.Update{update})
			assert.NoError(t, err, "Apply")
			expect.Equal(t, written, []string{filepath.Join(dir, othersFile)}, "the files that Apply wrote")
			expect.Contains(
				t,
				files.Read(t, filepath.Join(dir, othersFile)),
				`return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.1.0"}}`,
				"the literal of Other",
			)
		})

		t.Run("writes no file without an update", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			var written []string
			files.Unchanged(t, os.DirFS(dir), func() {
				var err error
				written, err = rewrite.Apply(dir, "Producer", nil)
				assert.NoError(t, err, "Apply")
			}, "the package of the cases")
			assert.Empty(t, written, "the files that Apply wrote")
		})

		t.Run("writes no file when a later update fails", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			failing := rewrite.Update{Pin: lintPin, Release: pin.Release{Version: "v1.3.0"}}
			failing.Pin.Field = []string{"Tools", "Missing"}
			var err error
			files.Unchanged(t, os.DirFS(dir), func() {
				_, err = rewrite.Apply(dir, "Producer", append(kinds[:1:1], failing))
			}, "the package of the cases")
			assert.ErrorIs(t, err, rewrite.ErrNotLiteral, "the error of Apply")
		})

		tests := []struct {
			name     string
			producer string
			give     rewrite.Update
			want     error
			text     string
		}{
			{
				name:     "returns ErrNoOptions for a producer without an Options method",
				producer: "Missing",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNoOptions,
				text:     "rewrite: the package has no Options method of the producer: Missing in " + dirMark,
			},
			{
				name:     "returns ErrNoOptions for a producer of a test file",
				producer: "Tested",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNoOptions,
				text:     "rewrite: the package has no Options method of the producer: Tested in " + dirMark,
			},
			{
				name:     "returns ErrNotLiteral for an Options method that returns a local variable",
				producer: "Local",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Tools " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for an Options method without a body",
				producer: "Bodiless",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Tools " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for an Options method without a statement",
				producer: "Empty",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Tools " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for an Options method that returns two results",
				producer: "Pair",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Tools " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for an Options method that ends in no return",
				producer: "Panicking",
				give:     rewrite.Update{Pin: lintPin},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Tools " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for a field that the literal leaves out",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Tools", "Missing")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the literal has " +
					"no field Missing",
			},
			{
				name:     "returns ErrNotLiteral for a value that a function computes",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Faults", "Computed")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the value is no " +
					"string literal",
			},
			{
				name:     "returns ErrNotLiteral for a value that a constant states",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Faults", "Constant")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the value is no " +
					"string literal",
			},
			{
				name:     "returns ErrNotLiteral for a literal that is no string",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Faults", "Number")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the value is no " +
					"string literal",
			},
			{
				name:     "returns ErrNotLiteral for a value that a receive returns",
				producer: "Producer",
				give: rewrite.Update{Pin: pin.Pin{
					Value:   option.Module("received@1.0.0"),
					Key:     "producer.faults.received",
					Version: "1.0.0",
					Field:   []string{"Faults", "Received"},
					Kind:    pin.KindModule,
				}},
				want: rewrite.ErrNotLiteral,
				text: "rewrite: producer.faults.received: rewrite: the value of the pin is no literal: the value " +
					"is no string literal",
			},
			{
				name:     "returns ErrNotLiteral for a field of a value that a function computes",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Faults", "Tools", "Lint")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the field Lint " +
					"is in no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for a key of a map",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(lintPin, "Faults", "Mapped", "lint")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.lint: rewrite: the value of the pin is no literal: the literal has " +
					"no field lint",
			},
			{
				name:     "returns ErrNotLiteral for a release binary whose literal has no keys",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkPin, "Faults", "Positional")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.check: rewrite: the value of the pin is no literal: the literal " +
					"has no field Version",
			},
			{
				name:     "returns ErrNotLiteral for an action without a commit",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkoutPin, "Faults", "Action")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.ci.actions.checkout: rewrite: the value of the pin is no literal: the " +
					"literal has no field Commit",
			},
			{
				name:     "returns ErrNotLiteral for a release binary without digests",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkPin, "Faults", "Bare")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.check: rewrite: the value of the pin is no literal: the literal " +
					"has no field SHA256",
			},
			{
				name:     "returns ErrNotLiteral for digests that a function computes",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkPin, "Faults", "Digests")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.check: rewrite: the value of the pin is no literal: the digests are " +
					"no composite literal",
			},
			{
				name:     "returns ErrNotLiteral for a digest without a platform",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkPin, "Faults", "Keyless")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.check: rewrite: the value of the pin is no literal: a digest has " +
					"no platform",
			},
			{
				name:     "returns ErrNotLiteral for a digest that a function computes",
				producer: "Producer",
				give:     rewrite.Update{Pin: fault(checkPin, "Faults", "Called")},
				want:     rewrite.ErrNotLiteral,
				text: "rewrite: producer.tools.check: rewrite: the value of the pin is no literal: the value is " +
					"no string literal",
			},
			{
				name:     "returns ErrMismatch for a literal with another value than the pin",
				producer: "Producer",
				give: rewrite.Update{Pin: pin.Pin{
					Value:   option.Module("example.com/lint/cmd/lint@v1.1.0"),
					Key:     "producer.tools.lint",
					Version: "v1.1.0",
					Field:   []string{"Tools", "Lint"},
					Kind:    pin.KindModule,
				}},
				want: rewrite.ErrMismatch,
				text: `rewrite: producer.tools.lint: rewrite: the literal differs from the pin: the literal is ` +
					`"example.com/lint/cmd/lint@v1.2.0", and the pin "example.com/lint/cmd/lint@v1.1.0"`,
			},
			{
				name:     "returns ErrMismatch for an action with another commit than the pin",
				producer: "Producer",
				give: rewrite.Update{Pin: pin.Pin{
					Value:   workflow.Action{Uses: "actions/checkout", Commit: nextCheckout, Release: "v7.0.1"},
					Key:     "producer.ci.actions.checkout",
					Version: "v7.0.1",
					Field:   []string{"CI", "Actions", "Checkout"},
					Kind:    pin.KindAction,
				}},
				want: rewrite.ErrMismatch,
				text: `rewrite: producer.ci.actions.checkout: rewrite: the literal differs from the pin: the ` +
					`literal is "` + checkout + `", and the pin "` + nextCheckout + `"`,
			},
			{
				name:     "returns ErrMismatch for a release binary with another version than the pin",
				producer: "Producer",
				give: rewrite.Update{Pin: pin.Pin{
					Value:   option.UV{Binary: option.Binary{Version: "0.9.0"}},
					Key:     "producer.tools.uv",
					Version: "0.9.0",
					Field:   []string{"Tools", "UV"},
					Kind:    pin.KindBinary,
				}},
				want: rewrite.ErrMismatch,
				text: `rewrite: producer.tools.uv: rewrite: the literal differs from the pin: the literal is ` +
					`"0.10.0", and the pin "0.9.0"`,
			},
			{
				name:     "returns ErrMismatch for a digest that the pin does not state",
				producer: "Producer",
				give: rewrite.Update{
					Pin: pin.Pin{
						Value: option.UV{Binary: option.Binary{
							SHA256:  map[option.Platform]string{option.LinuxAMD64: otherDigest},
							Version: "0.10.0",
						}},
						Key:     "producer.tools.uv",
						Version: "0.10.0",
						Field:   []string{"Tools", "UV"},
						Kind:    pin.KindBinary,
					},
					Release: pin.Release{Digests: map[option.Platform]string{option.LinuxAMD64: nextLinux}},
				},
				want: rewrite.ErrMismatch,
				text: "rewrite: producer.tools.uv: rewrite: the literal differs from the pin: the release has no " +
					"one digest for the platforms of " + uvLinux,
			},
			{
				name:     "returns ErrMismatch for a release without the digest of a platform",
				producer: "Producer",
				give: rewrite.Update{
					Pin:     uvPin,
					Release: pin.Release{Digests: map[option.Platform]string{option.LinuxAMD64: nextLinux}},
				},
				want: rewrite.ErrMismatch,
				text: "rewrite: producer.tools.uv: rewrite: the literal differs from the pin: the release has no " +
					"one digest for the platforms of " + uvDarwin,
			},
			{
				name:     "returns ErrMismatch for a release with two digests for the platforms of one digest",
				producer: "Producer",
				give: rewrite.Update{
					Pin: checkPin,
					Release: pin.Release{
						Digests: map[option.Platform]string{
							option.LinuxAMD64: nextCheck,
							option.LinuxARM64: otherDigest,
						},
					},
				},
				want: rewrite.ErrMismatch,
				text: "rewrite: producer.tools.check: rewrite: the literal differs from the pin: the release has " +
					"no one digest for the platforms of " + checkLinux,
			},
			{
				name:     "returns ErrMismatch for a pin of a release binary whose value is none",
				producer: "Producer",
				give: rewrite.Update{
					Pin: pin.Pin{Value: option.Module("uv@0.10.0"), Key: "producer.tools.uv", Kind: pin.KindBinary},
				},
				want: rewrite.ErrMismatch,
				text: "rewrite: producer.tools.uv: rewrite: the literal differs from the pin: the pin is no " +
					"release binary",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := copied(t)
				var err error
				files.Unchanged(t, os.DirFS(dir), func() {
					_, err = rewrite.Apply(dir, tt.producer, []rewrite.Update{tt.give})
				}, "the package of the cases")
				assert.ErrorIs(t, err, tt.want, "the error of Apply")
				assert.Equal(t, err.Error(), strings.ReplaceAll(tt.text, dirMark, dir), "the text of the error")
			})
		}

		t.Run("returns the error of a directory that does not exist", func(t *testing.T) {
			t.Parallel()
			_, err := rewrite.Apply(filepath.Join(t.TempDir(), "missing"), "Producer", kinds)
			assert.ErrorIs(t, err, fs.ErrNotExist, "the error of Apply")
		})

		t.Run("returns the error of a file that does not read", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			unread := filepath.Join(dir, "directory.go")
			assert.NoError(t, os.Mkdir(unread, 0o755), "Mkdir of directory.go")
			_, err := rewrite.Apply(dir, "Producer", kinds)
			assert.HasPrefix(t, err.Error(), "rewrite: read "+unread+": ", "the error of Apply")
		})

		t.Run("returns the error of a file that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			broken := filepath.Join(dir, "broken.go")
			assert.NoError(
				t,
				os.WriteFile(broken, []byte("package producer\n\nfunc (\n"), 0o600),
				"WriteFile of broken.go",
			)
			_, err := rewrite.Apply(dir, "Producer", kinds)
			assert.HasPrefix(t, err.Error(), "rewrite: "+broken+":", "the error of Apply")
		})

		t.Run("returns the files that it wrote before a file that does not write", func(t *testing.T) {
			t.Parallel()
			dir := copied(t)
			assert.NoError(t, os.Chmod(filepath.Join(dir, varsFile), 0o444), "Chmod of vars.go")
			written, err := rewrite.Apply(dir, "Producer", kinds)
			assert.ErrorIs(t, err, fs.ErrPermission, "the error of Apply")
			expect.Equal(t, written, []string{filepath.Join(dir, producerFile)}, "the files that Apply wrote")
			expect.Contains(
				t,
				files.Read(t, filepath.Join(dir, producerFile)),
				`"example.com/lint/cmd/lint@v1.10.0"`,
				"the literal of the linter",
			)
		})
	})
}

// copied returns a directory of the test t with a copy of the package of the cases.
func copied(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.CopyFS(dir, os.DirFS(source)), "CopyFS of the package of the cases")
	return dir
}

// fault returns p with the field names field, a field of the literal of the producer Producer of
// the cases that fails the rewrite.
func fault(p pin.Pin, field ...string) pin.Pin {
	p.Field = field
	return p
}
