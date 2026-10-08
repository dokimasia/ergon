// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package producer

import "go.dokimi.dev/ergon/core/language"

// Tested is a producer of a test file.
type Tested struct{}

// Options returns the baseline of Tested.
func (Tested) Options() language.Options {
	return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.2.0"}}
}
