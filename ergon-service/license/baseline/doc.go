// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of the license files of ergon init: LICENSE, and NOTICE for
// Apache-2.0.
//
// [Producer] renders both files as managed files, from the texts that
// [go.dokimi.dev/ergon/service/license.Text] returns for the answers of ergon init and the section
// license of .ergon.yaml. A license text has no comment, so the lock alone records the files. The
// templates under templates/managed/ write the texts that [Producer.Data] computes.
//
// # Options
//
// The section license of .ergon.yaml is [go.dokimi.dev/ergon/service/license.Config]: ergon init
// writes its owner and its license from the answers, and the repository sets the parameters of
// BUSL-1.1, the comment styles and the paths without a header. ergon init fails for BUSL-1.1 until
// the section sets each parameter.
//
// # Workflows
//
// [Producer.Contribution] returns the job license of ci.yml, which runs ergon license check on the
// Linux runner of the section github.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/workflow] and [go.dokimi.dev/ergon/service/license]. internal/cli of the
// root module imports it.
package baseline
