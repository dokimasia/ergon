// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
)

// fakeGPG is the name of the fake gpg, a copy of the test binary that TestMain runs as gpg.
const fakeGPG = "gpg"

// failPrefix starts a file that makes the fake gpg fail. The first line of such a file is
// fail <status>, and the lines after it are the message of gpg. The fake fails on such a file to
// sign, and on such a signature to verify.
const failPrefix = "fail "

// validSig and validSigWithoutPrimary are the status lines of gpg for a valid signature of the
// cases, with the primary key and without it.
const (
	validSig = "[GNUPG:] NEWSIG\n[GNUPG:] VALIDSIG " + signingKey + " 2026-10-29 1793283600 0 4 0 1 10 00 " +
		primaryKey + "\n"
	validSigWithoutPrimary = "[GNUPG:] VALIDSIG " + signingKey + " 2026-10-29 1793283600 0 4 0 17 10 00\n"
)

// signingKey is the fingerprint of the signing subkey of the cases, and primaryKey is the
// fingerprint of its primary key.
const (
	signingKey = "0123456789ABCDEF0123456789ABCDEF01234567"
	primaryKey = "89ABCDEF0123456789ABCDEF0123456789ABCDEF"
)

// gpgMain runs the test binary as the fake gpg when its name is gpg, and reports whether it did,
// with the exit status of the fake. With --detach-sign the fake writes its arguments and its
// standard input to the file of --output, and writes "gpg: signing" to its standard error. With
// --verify it writes the signature to its standard output, because the signature of a case contains
// the status lines of gpg. A file that starts with failPrefix makes it fail.
func gpgMain(args []string) (int, bool) {
	if strings.TrimSuffix(filepath.Base(args[0]), ".exe") != fakeGPG {
		return 0, false
	}
	rest := args[1:]
	if i := slices.Index(rest, "--verify"); i >= 0 && i+1 < len(rest) {
		data, err := os.ReadFile(rest[i+1])
		if err != nil {
			return 2, true
		}
		if status, failed := failure(data); failed {
			return status, true
		}
		_, _ = os.Stdout.Write(data)
		return 0, true
	}
	i := slices.Index(rest, "--output")
	if !slices.Contains(rest, "--detach-sign") || i < 0 || i+1 >= len(rest) {
		return 2, true
	}
	data, err := os.ReadFile(rest[len(rest)-1])
	if err != nil {
		return 2, true
	}
	if status, failed := failure(data); failed {
		return status, true
	}
	stdin, _ := io.ReadAll(os.Stdin)
	fmt.Fprintln(os.Stderr, "gpg: signing")
	if err := os.WriteFile(rest[i+1], []byte(strings.Join(rest, " ")+"\n"+string(stdin)), 0o600); err != nil {
		return 2, true
	}
	return 0, true
}

// failure reports whether data starts with failPrefix, and returns its status after it writes the
// message of the lines after the first to the standard error.
func failure(data []byte) (int, bool) {
	first, message, _ := strings.Cut(string(data), "\n")
	code, found := strings.CutPrefix(first, failPrefix)
	if !found {
		return 0, false
	}
	status, err := strconv.Atoi(code)
	if err != nil {
		return 2, true
	}
	fmt.Fprint(os.Stderr, message)
	return status, true
}

// installGPG copies the test binary as gpg, with .exe on Windows, into a new directory, and puts the
// directory first on the PATH of the process, from which exec resolves gpg. It returns the
// directory, and the error of the copy.
func installGPG() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "ergon-release-gpg-")
	if err != nil {
		return "", err
	}
	name := fakeGPG
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o755); err != nil {
		return dir, err
	}
	return dir, os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGPG(t *testing.T) {
	t.Parallel()

	t.Run("Sign", func(t *testing.T) {
		t.Parallel()

		t.Run("runs gpg --local-user --detach-sign --output on its terminal", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{"sums": files.Text("checksums\n")})
			sums, signature := filepath.Join(dir, "sums"), filepath.Join(dir, "sums.sig")
			var stderr strings.Builder
			g := release.GPG{Terminal: vcs.Terminal{Stdin: strings.NewReader("pin\n"), Stderr: &stderr}}
			assert.NoError(t, g.Sign(t.Context(), signingKey, sums, signature), "Sign")
			want := "--local-user " + signingKey + " --detach-sign --output " + signature + " " + sums + "\npin\n"
			files.HasContent(t, signature, want, "the arguments and the standard input of gpg")
			assert.Equal(t, stderr.String(), "gpg: signing\n", "the standard error of gpg")
		})

		t.Run("returns the error of gpg with the name of the file", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{"sums": files.Text(failPrefix + "2\ngpg: no secret key\n")})
			var stderr strings.Builder
			g := release.GPG{Terminal: vcs.Terminal{Stderr: &stderr}}
			err := g.Sign(t.Context(), signingKey, filepath.Join(dir, "sums"), filepath.Join(dir, "sums.sig"))
			exit := assert.ErrorAs[*exec.ExitError](t, err, "Sign")
			expect.Equal(t, exit.ExitCode(), 2, "the exit status of gpg")
			expect.Contains(t, err.Error(), "gpg --detach-sign of sums", "the error")
			expect.Equal(t, stderr.String(), "gpg: no secret key\n", "the standard error of gpg")
		})

		t.Run("runs the program of Program", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{"sums": files.Text("checksums\n")})
			g := release.GPG{Program: filepath.Join(dir, "absent")}
			err := g.Sign(t.Context(), signingKey, filepath.Join(dir, "sums"), filepath.Join(dir, "sums.sig"))
			assert.HasError(t, err, "Sign")
			files.Absent(t, filepath.Join(dir, "sums.sig"), "the signature")
		})
	})

	t.Run("Verify", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			give      string
			wantError string
			want      release.Signature
		}{
			{
				name: "returns the keys and the algorithm of the line VALIDSIG",
				give: validSig,
				want: release.Signature{Fingerprint: signingKey, Primary: primaryKey, Algorithm: 1},
			},
			{
				name: "returns no primary key for a line VALIDSIG without one",
				give: validSigWithoutPrimary,
				want: release.Signature{Fingerprint: signingKey, Algorithm: 17},
			},
			{
				name:      "returns ErrVerify for a signature that gpg rejects",
				give:      failPrefix + "1\ngpg: BAD signature\n",
				wantError: "gpg --verify of sums",
			},
			{
				name:      "returns ErrVerify for an output without the line VALIDSIG",
				give:      "[GNUPG:] NEWSIG\n[GNUPG:] VALIDSIG " + signingKey + "\n",
				wantError: "gpg --verify of sums wrote no line VALIDSIG",
			},
			{
				name:      "returns ErrVerify for an algorithm that is not a number",
				give:      "[GNUPG:] VALIDSIG " + signingKey + " 2026-10-29 1793283600 0 4 0 rsa 10 00\n",
				wantError: `the algorithm "rsa"`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := files.Workspace(t, files.Tree{
					"sums":     files.Text("checksums\n"),
					"sums.sig": files.Text(tt.give),
				})
				g := release.GPG{}
				got, err := g.Verify(t.Context(), filepath.Join(dir, "sums"), filepath.Join(dir, "sums.sig"))
				if tt.wantError != "" {
					assert.ErrorIs(t, err, release.ErrVerify, "Verify")
					assert.Contains(t, err.Error(), tt.wantError, "the error")
					return
				}
				assert.NoError(t, err, "Verify")
				assert.Equal(t, got, tt.want, "the key of the signature")
			})
		}

		t.Run("writes the messages of gpg to its terminal", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{
				"sums":     files.Text("checksums\n"),
				"sums.sig": files.Text(failPrefix + "1\ngpg: Can't check signature: No public key\n"),
			})
			var stderr strings.Builder
			g := release.GPG{Terminal: vcs.Terminal{Stderr: &stderr}}
			_, err := g.Verify(t.Context(), filepath.Join(dir, "sums"), filepath.Join(dir, "sums.sig"))
			assert.ErrorIs(t, err, release.ErrVerify, "Verify")
			assert.Equal(t, stderr.String(), "gpg: Can't check signature: No public key\n", "the standard error of gpg")
		})
	})
}
