// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/buildinfo"
	"go.dokimi.dev/ergon/internal/cli"
)

func main() {
	os.Exit(run())
}

// run runs the command line of the process and returns its exit status. SIGINT and SIGTERM cancel
// the context of the command. main exits with the status after the deferred calls of run have
// returned.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	p := cli.Process{
		Getwd:     os.Getwd,
		Now:       time.Now,
		CacheDir:  os.UserCacheDir,
		Random:    rand.Reader,
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Transport: http.DefaultTransport,
		Platform:  option.Platform(runtime.GOOS + "/" + runtime.GOARCH),
		Args:      os.Args[1:],
		Env:       os.Environ(),
	}
	info := buildinfo.Read()
	return cli.Run(ctx, &p, app.Register, cli.Version{Release: info.Version, Full: info.String()})
}
