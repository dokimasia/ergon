// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// ErrGit is the error for a command of git that does not run or that ends with an error, such as
// git ls-files in a directory outside a working tree.
var ErrGit = errors.New("vcs: git failed")

// Terminal is the terminal of a command of git whose programs prompt for a PIN or a touch: the
// signing program of git tag for those of a signing key, and the ssh of git push for those of the
// key of the remote. ssh-keygen reads the PIN of a key from its standard input only when that is a
// terminal, and from the program of SSH_ASKPASS otherwise. The zero value is no terminal: git and
// its programs read the null device, and the error of a failed command states the standard error of
// git.
//
// # Concurrency
//
// Concurrent commands that share a Terminal interleave their prompts on it.
type Terminal struct {
	// Stdin is the standard input of git and of the programs that it starts, or nil for the null
	// device. git passes an *os.File, such as [os.Stdin], to its programs as their own standard
	// input, so a terminal on it is a terminal for ssh-keygen.
	Stdin io.Reader

	// Stderr is the standard error of git and of the programs that it starts, such as the request
	// of ssh to touch a key, or nil for none. A command with a Stderr states no standard error in
	// its error, because git wrote it to Stderr.
	Stderr io.Writer
}

// run runs git with args in dir and returns its standard output. It returns an error that wraps
// [ErrGit] and states args and the standard error of git when git does not run, when it ends with
// an error, and when ctx ends before git does.
func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return runEnv(ctx, dir, nil, Terminal{}, args...)
}

// runEnv runs git as [run] does, with the variables of env, each NAME=value, added to the
// environment of the process, and with the standard input and the standard error of t. Its error
// states the standard error of git when t has no Stderr.
func runEnv(ctx context.Context, dir string, env []string, t Terminal, args ...string) ([]byte, error) {
	all := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr bytes.Buffer
	cmd.Stdin, cmd.Stderr = t.Stdin, t.Stderr
	if t.Stderr == nil {
		cmd.Stderr = &stderr
	}
	out, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("%w: git %s: %w", ErrGit, strings.Join(all, " "), err)
		if text := strings.TrimSpace(stderr.String()); text != "" {
			err = fmt.Errorf("%w: %s", err, text)
		}
		return nil, err
	}
	return out, nil
}

// paths returns the paths of out, the output of git with -z: one path after each NUL byte. It
// returns them sorted, each once, and nil for an empty out.
func paths(out []byte) []string {
	list := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if list[0] == "" {
		return nil
	}
	slices.Sort(list)
	return slices.Compact(list)
}
