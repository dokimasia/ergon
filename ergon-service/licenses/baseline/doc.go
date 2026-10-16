// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of the license files of ergon init: LICENSE, and NOTICE for
// Apache-2.0, at the root and in each directory under a license of its own.
//
// [Producer] renders the files as managed files, from the texts that
// [go.dokimi.dev/ergon/service/licenses.Text] returns for the answers of ergon init and the section
// license of .ergon.yaml. A license text has no comment, so the lock alone records the files. The
// templates under templates/managed/ write the texts that [Producer.Data] computes for the root.
// [Producer.Files] returns the files of each directory of the section, whose paths no template tree
// can mirror.
//
// # Options
//
// The section license of .ergon.yaml is [go.dokimi.dev/ergon/service/licenses.Config]: ergon init
// writes its owner and its license from the answers, and the repository sets the parameters of
// BUSL-1.1, the directories under a license of their own, the comment styles and the paths without
// a header. ergon init new also writes the parameters of BUSL-1.1 from its flags. A parameter may be
// empty, and its line of LICENSE then has its label alone.
//
// # Headers
//
// [Producer.Annotate] adds the license header of the section to each file that ergon init creates
// for the repository to maintain: the seeded files, and .ergon.yaml when the repository has none.
// It writes the header that ergon license fix writes, with the year of the answers, so a new
// repository passes ergon license check.
//
// # Workflows
//
// [Producer.Contribution] returns the job license of ci.yml, which runs ergon license check on the
// Linux runner of the section github.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/workflow] and [go.dokimi.dev/ergon/service/licenses]. internal/cli of
// the root module imports it.
package baseline
