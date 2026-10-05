// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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
	return cli.Run(ctx, os.Args[1:], app.Register, buildinfo.Full(), os.Stdout, os.Stderr)
}
