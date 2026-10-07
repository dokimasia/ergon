// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of PHP of ergon init: the configuration of PHPStan, the
// fragments of PHP of the shared files, the section php of .ergon.yaml, and the part of PHP of the
// workflows.
//
// [Producer] renders phpstan.dist.neon as a managed file from templates/managed/, and its fragments
// of .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-php, lint-php, test-php and audit-php, and check-php, which
// requires the targets of the steps that the key check of the section names. PHPStan and
// PHP-CS-Fixer run through ergon tool run, which installs the Composer packages of the section
// together into one project of its own, apart from the composer.json of the repository. PHPUnit
// runs from the vendor directory of the repository, and audit-php runs composer audit over
// composer.lock.
//
// # Options
//
// [Options] is the section php: the tools of the targets, the paths that PHP-CS-Fixer and PHPStan
// check, the steps of the gate, the options of test-php, and the key ci of the job check-php.
// [Tools.Validate] requires the package of each tool of the baseline. [Options.Contribution]
// returns the job check-php and the updates of the Composer packages.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
