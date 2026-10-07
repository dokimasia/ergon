// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/lang/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrorPrefix(t *testing.T) {
	t.Parallel()

	t.Run("ErrorPrefix", func(t *testing.T) {
		t.Parallel()

		t.Run("reports each constant text of errors.New without the name of its package", func(t *testing.T) {
			t.Parallel()
			analysistest.Run(t, analysistest.TestData(), analysis.ErrorPrefix(none), "clock", "other")
		})

		t.Run("is named errorprefix", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, analysis.ErrorPrefix(none).Name, "errorprefix", "the name of the analyzer")
		})
	})
}

// none is the skip of the cases that skips no package.
func none(string) bool {
	return false
}
