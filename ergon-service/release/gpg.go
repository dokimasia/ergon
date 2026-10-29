// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"cmp"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/service/vcs"
)

// gpgProgram is the program of gpg of the PATH, which a [GPG] without a program runs.
const gpgProgram = "gpg"

// The status line of gpg for a valid signature is [GNUPG:] VALIDSIG, followed by the fingerprint of
// the key, the date and the time of the signature, its expiry, its version, a reserved field, the
// public key algorithm, the hash algorithm, the class of the signature, and the fingerprint of the
// primary key. gpg may leave out the primary key. GnuPG documents the fields in doc/DETAILS.
const (
	statusPrefix = "[GNUPG:]"
	validSig     = "VALIDSIG"

	// validSigFields is the number of fields of the line without the primary key.
	validSigFields = 11

	// The positions of the fields of the line that a [Signature] reads.
	fingerprintField = 2
	algorithmField   = 8
	primaryField     = 11
)

// GPG signs files and verifies signatures with gpg, whose agent asks for the PIN of a key on the
// terminal of a person, such as the PIN and the touch of a smart card. The zero value runs the gpg
// of the PATH without a terminal.
//
// # Concurrency
//
// A GPG is safe for concurrent use, but concurrent prompts interleave on its terminal.
type GPG struct {
	// Terminal is the terminal of gpg, with its standard input and the standard error that receives
	// its messages.
	Terminal vcs.Terminal

	// Program is the program of gpg, or empty for gpg of the PATH.
	Program string
}

var _ Signer = GPG{}

// Sign signs the file at path with the key of the fingerprint key, and writes the binary detached
// signature to the path signature. It runs gpg --local-user key --detach-sign --output signature
// path on the terminal of g. It returns the error of gpg with the base name of path.
func (g GPG) Sign(ctx context.Context, key, path, signature string) error {
	program := cmp.Or(g.Program, gpgProgram)
	cmd := exec.CommandContext(ctx, program, "--local-user", key, "--detach-sign", "--output", signature, path)
	cmd.Stdin, cmd.Stderr = g.Terminal.Stdin, g.Terminal.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("release: gpg --detach-sign of %s: %w", filepath.Base(path), err)
	}
	return nil
}

// Verify verifies the detached signature at signature of the file at path with gpg --batch
// --status-fd 1 --verify, and returns the key of its VALIDSIG line. gpg writes its messages to the
// standard error of the terminal of g. It returns an error that wraps [ErrVerify] for a signature
// that gpg rejects, and for an output without a VALIDSIG line of all its fields, each with the base
// name of path.
func (g GPG) Verify(ctx context.Context, path, signature string) (Signature, error) {
	program := cmp.Or(g.Program, gpgProgram)
	cmd := exec.CommandContext(ctx, program, "--batch", "--status-fd", "1", "--verify", signature, path)
	cmd.Stderr = g.Terminal.Stderr
	out, err := cmd.Output()
	if err != nil {
		return Signature{}, fmt.Errorf("%w: gpg --verify of %s: %w", ErrVerify, filepath.Base(path), err)
	}
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < validSigFields || fields[0] != statusPrefix || fields[1] != validSig {
			continue
		}
		algorithm, err := strconv.Atoi(fields[algorithmField])
		if err != nil {
			return Signature{}, fmt.Errorf("%w: gpg --verify of %s: the algorithm %q", ErrVerify, filepath.Base(path),
				fields[algorithmField])
		}
		s := Signature{Fingerprint: fields[fingerprintField], Algorithm: algorithm}
		if len(fields) > primaryField {
			s.Primary = fields[primaryField]
		}
		return s, nil
	}
	return Signature{}, fmt.Errorf("%w: gpg --verify of %s wrote no line %s", ErrVerify, filepath.Base(path), validSig)
}
